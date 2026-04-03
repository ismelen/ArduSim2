import { useEffect, useRef, useState } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';

/**
 * METERS_PER_DEGREE_LAT: Approximate meters per degree of latitude.
 * 111,319.5m is a standard constant for WGS84 at most latitudes.
 */
const METERS_PER_DEGREE_LAT = 111319.5;

export interface TelemetryData {
    uav_id: string;
    payload: {
        position: { lat: number; lon: number; alt: number; heading: number };
        speed: { vx: number; vy: number; vz: number };
        battery: number;
        status: string;
        flight_mode: string;
        time_boot_ms: number;
    };
}

export interface UAVState {
    id: string;
    lat: number;
    lon: number;
    alt: number;
    heading: number;
    vx: number; // North (m/s)
    vy: number; // East (m/s)
    vz: number; // Down (m/s)
    lastUpdate: number;
    battery: number;
    status: string;
    flight_mode: string;
}

/**
 * useTelemetry hook manages real-time UAV telemetry state.
 * It listens for 'telemetry' events from Wails and performs
 * dead-reckoning extrapolation at 60 FPS for smooth UI movement.
 */
export function useTelemetry() {
    const uavsRef = useRef<Record<string, UAVState>>({});
    const [predictedUavs, setPredictedUavs] = useState<Record<string, UAVState>>({});

    useEffect(() => {
        // Subscribe to telemetry events from Go backend
        const unsubscribe = EventsOn('telemetry', (msg: TelemetryData) => {
            const now = performance.now();
            const newState: UAVState = {
                id: msg.uav_id,
                lat: msg.payload.position.lat,
                lon: msg.payload.position.lon,
                alt: msg.payload.position.alt,
                heading: msg.payload.position.heading,
                vx: msg.payload.speed.vx,
                vy: msg.payload.speed.vy,
                vz: msg.payload.speed.vz,
                lastUpdate: now,
                battery: msg.payload.battery,
                status: msg.payload.status,
                flight_mode: msg.payload.flight_mode,
            };
            uavsRef.current[msg.uav_id] = newState;
        });

        return () => {
            if (unsubscribe) unsubscribe();
        };
    }, []);

    useEffect(() => {
        let animationFrameId: number;

        const updatePositions = () => {
            const now = performance.now();
            const nextStates: Record<string, UAVState> = {};

            Object.entries(uavsRef.current).forEach(([id, state]) => {
                const dt = (now - state.lastUpdate) / 1000; // time in seconds since last UDP packet

                // Prediction logic (Dead Reckoning)
                // New Position = Received Position + (Velocity * dt)
                
                // 1. Latitude change (vx = North)
                const dLat = (state.vx * dt) / METERS_PER_DEGREE_LAT;
                
                // 2. Longitude change (vy = East). Depends on latitude.
                const latRad = (state.lat * Math.PI) / 180;
                const dLon = (state.vy * dt) / (METERS_PER_DEGREE_LAT * Math.cos(latRad));

                nextStates[id] = {
                    ...state,
                    lat: state.lat + dLat,
                    lon: state.lon + dLon,
                };
            });

            setPredictedUavs(nextStates);
            animationFrameId = requestAnimationFrame(updatePositions);
        };

        animationFrameId = requestAnimationFrame(updatePositions);
        return () => cancelAnimationFrame(animationFrameId);
    }, []);

    return predictedUavs;
}
