# Application (Central Orchestrator)

Este componente actúa como el **núcleo de control** de ArduSim2. Sus funciones principales son orquestar las sugerencias de movimiento de múltiples algoritmos y servir de puente para la telemetría del dron.

## Funciones Principales

### 1. Mezclador de Movimiento (Movement Mixer)
El `application` centraliza todas las "sugerencias" (`Suggestions`) enviadas por los algoritmos de vuelo. Utiliza una ventana de tiempo configurable (`mix_window_ms`) para agrupar estas sugerencias y decidir la acción final a enviar al dron.

-   **Prioridad Estructural**: Si se recibe una sugerencia "estructural" (como `Land`, `RTL` o `BRAKE`), ésta anula inmediatamente cualquier otra sugerencia de movimiento vectorial.
-   **Mezcla Vectorial**: Si se reciben múltiples sugerencias de movimiento (como `MoveToPosition`) en la misma ventana, el sistema calcula un promedio de las coordenadas (Latitud, Longitud, Altitud) para suavizar y unificar el comportamiento.

### 2. Puente de Telemetría (Telemetry Bridge)
El componente mantiene un enlace directo con el controlador del UAV para recibir su telemetría. Cada paquete recibido se retransmite automáticamente al **Communication Module** bajo un tópico unificado, permitiendo que todos los algoritmos suscritos tengan acceso a los datos de vuelo en tiempo real.

## Uso

Para ejecutar el orquestador:

```bash
go run cmd/main.go <config.json>
```

## Configuración (`config.json`)

El archivo de configuración define los parámetros de red y los tópicos de comunicación:

| Campo | Descripción |
| :--- | :--- |
| `broker_ip` | Dirección IP del Communication Module (ej: `communication_module`). |
| `broker_port` | Puerto UDP del Communication Module (por defecto `3400`). |
| `uav_controller_ip` | Dirección IP del controlador del dron (`uav_controller`). |
| `uav_controller_port`| Puerto para enviar comandos al dron (ej: `9876`). |
| `uav_telemetry_port` | Puerto local para recibir telemetría del dron (ej: `3500`). |
| `telemetry_topic` | Tópico donde se publica la telemetría unificada (ej: `uav/telemetry`). |
| `suggestions_topic` | Tópico donde el mixer escucha sugerencias de los algoritmos (ej: `uav/suggestions`). |
| `global_commands` | Tópico para emitir órdenes globales de parada a todos los algoritmos. |
| `mix_window_ms` | Tamaño de la ventana (en ms) para agrupar y mezclar sugerencias. |

## Arquitectura de Comunicación

### Entrada (Suscripciones)
El componente se comunica con el Broker UDP para suscribirse a los siguientes canales:
-   **Sugerencias**: Recibe las intenciones de vuelo de los algoritmos.
-   **Telemetría UAV**: Recibe los datos crudos del dron para su procesamiento o retransmisión.

### Salida (Publicaciones)
-   **Telemetría Unificada**: Publica los datos del dron para consumo global de los algoritmos.
-   **Comandos UAV**: Envía las órdenes finales mezcladas directamente al controlador del dron.
-   **Parada Global**: Si se detecta un evento crítico (Land/RTL), emite un comando de parada en el canal `global_commands` para sincronizar a todos los agentes.

---
*Este componente es vital para el funcionamiento seguro de ArduSim2, garantizando que el dron no reciba órdenes contradictorias de múltiples fuentes simultáneamente.*
