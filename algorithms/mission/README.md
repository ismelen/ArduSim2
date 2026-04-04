# Mission Algorithm

Este algoritmo permite la ejecución de misiones de vuelo autónomas para UAVs a partir de un archivo de definición de ruta en formato **KML**. El sistema se comunica mediante un protocolo **UDP con mensajes JSON**.

## Funcionamiento

El algoritmo lee una serie de waypoints desde un archivo KML y guía al dron a través de ellos de forma secuencial.

1.  **Inicio/Reinicio**: Tras recibir el comando de inicio, el dron se arma, cambia a modo `GUIDED` y realiza un despegue automático.
2.  **Ejecución**: Una vez alcanzada la altitud de despegue, el dron se dirige al primer waypoint de la misión.
3.  **Progreso**: El algoritmo monitorea la posición del dron mediante telemetría. Cuando el dron está dentro del radio de alcance configurado de un waypoint, se considera alcanzado y se envía la orden para ir al siguiente.
4.  **Finalización**: Al alcanzar el último waypoint, el dron realiza un aterrizaje automático (`Land`).

## Uso

Para ejecutar el algoritmo desde la línea de comandos:

```bash
go run cmd/main.go <config.json>
```

El archivo `<config.json>` debe contener los parámetros de red, los tópicos de comunicación y la ruta al archivo de misión KML.

### Ejemplo de `config.json`
```json
{
  "mission_file": "/path/to/mission.kml",
  "broker_ip": "127.0.0.1",
  "broker_port": 3400,
  "subscription_topic": "algo/mission",
  "publish_topic": "uav/1/cmd",
  "telemetry_topic": "uav/1/telemetry",
  "distance_to_waypoint_reached": 5.0,
  "minimum_waypoint_relative_altitude": 10.0,
  "waypoints_relative_altitude": 15.0
}
```

### Campos de Configuración

| Campo | Tipo | Descripción |
| :--- | :--- | :--- |
| `mission_file` | `string` | Ruta al archivo `.kml` con la misión. |
| `broker_ip` | `string` | IP del Communication Module (broker UDP). |
| `broker_port` | `int` | Puerto del Communication Module (por defecto 3400). |
| `subscription_topic` | `string` | Tópico para recibir comandos (ej: `algo/mission`). |
| `publish_topic` | `string` | Tópico para enviar órdenes al dron (ej: `uav/suggestions`). |
| `telemetry_topic` | `string` | Tópico para recibir telemetría (ej: `uav/telemetry`). |
| `distance_to_waypoint_reached` | `number` | Distancia (m) para considerar un punto como "alcanzado". |
| `minimum_waypoint_relative_altitude` | `number` | Altitud mínima de despegue (m). |
| `waypoints_relative_altitude` | `number` | Altitud de crucero entre waypoints (m). |

## Integración con el Communication Module

Este algoritmo está diseñado para integrarse con el **Communication Module** de ArduSim2, el cual actúa como un broker de mensajes UDP.

-   **Puerto del Broker**: `3400` (UDP).
-   **Modelo de Comunicación**: Pub/Sub basado en tópicos (estilo MQTT).

### Registro de Canales (Suscripciones)
Al iniciar, el algoritmo envía mensajes especiales al broker para registrar sus intereses en los canales de entrada. Estos mensajes utilizan el tópico reservado `$subscribe`.

1.  **Canal de Comandos**: Se suscribe a `subscription_topic` (ej: `algo/mission`) para recibir órdenes de control.
2.  **Canal de Telemetría**: Se suscribe a `telemetry_topic` (ej: `uav/1/telemetry`) para recibir la posición en tiempo real del dron.

**Ejemplo de registro enviado al broker:**
```json
{
  "topic": "$subscribe",
  "subscribe_to": "algo/mission"
}
```

### Canales de Publicación
Cuando el algoritmo decide una acción (ej: ir a un punto), publica un mensaje JSON en el `publish_topic` (ej: `uav/1/cmd`). El broker se encarga de redirigir este mensaje a los contenedores interesados (normalmente el `uav_controller`).

---

## Comunicación (UDP JSON)

### Envío de Mensajes al Algoritmo
Para controlar el algoritmo, se deben enviar mensajes JSON al puerto y dirección configurados. El formato requiere un campo `topic` para la suscripción interna del broker UDP.

#### Comandos de Control (`subscription_topic`)
| Comando | Acción |
| :--- | :--- |
| `start` | Inicia la misión o la reanuda si estaba en pausa. |
| `pause` | Pausa la misión (pone el dron en modo `BRAKE`). |
| `rtl` | Ordena al dron volver a casa (`Return To Launch`). |
| `emergency_land` | Ordena un aterrizaje inmediato en la posición actual. |

**Ejemplo de mensaje para iniciar:**
```json
{
  "topic": "algo/mission",
  "command": "start"
}
```

**Ejemplo de mensaje para pausa:**
```json
{
  "topic": "algo/mission",
  "command": "pause"
}
```

### Mensajes de Telemetría Requeridos (`telemetry_topic`)
El algoritmo necesita recibir periódicamente la posición del dron:
```json
{
  "topic": "uav/1/telemetry",
  "lat": 39.4816,
  "lon": -0.3492,
  "relative_alt": 10.5
}
```

## Acciones Enviadas por el Algoritmo (`publish_topic`)

El algoritmo envía las siguientes órdenes al controlador del dron:

-   `{"endpoint": "Arm"}`: Arma los motores.
-   `{"endpoint": "SetFlightmode", "flightmode": "GUIDED" | "BRAKE" | "RTL"}`: Cambia el modo de vuelo.
-   `{"endpoint": "Takeoff", "altitude": <valor>}`: Despegue a una altitud específica.
-   `{"endpoint": "MoveToPosition", "latitude": <lat>, "longitude": <lon>, "altitude": <alt>}`: Movimiento a un punto geográfico.
-   `{"endpoint": "Land"}`: Aterrizaje.

## Estados del Algoritmo
-   **IDLE**: Esperando comando de inicio.
-   **TAKEOFF**: Realizando el despegue inicial.
-   **FLYING**: Navegando entre waypoints.
-   **PAUSED**: Misión detenida temporalmente (modo BRAKE).
-   **LANDING**: Misión terminada o abortada, en proceso de aterrizaje.
