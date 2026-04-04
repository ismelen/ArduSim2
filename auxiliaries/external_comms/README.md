# External Communications Bridge (Auxiliary Service)

Este servicio actúa como un **puente de comunicación** (Gateway Bridge) para un UAV dentro del ecosistema ArduSim2. Su función principal es interconectar el **Broker Local** (utilizado por los componentes internos del UAV, como el sistema de telemetría o algoritmos de misión) con un **Simulador de Red Externo (NetSim)** compartido por todo el enjambre.

## Función Principal

El bridge realiza una traducción bidireccional de mensajes:
1.  **De Interno a Externo (Local -> Swarm):** Escucha mensajes en el broker local del UAV y los encapsula para enviarlos al Simulador de Red global a través de UDP.
2.  **De Externo a Interno (Swarm -> Local):** Recibe paquetes del Simulador de Red distribuidos por otros UAVs y los publica en los tópicos correspondientes del broker local para que el UAV "escuche" a sus compañeros.

## Arquitectura de Comunicación

### 1. Broker Local (Docker/Interno)
Se conecta mediante UDP a un broker de mensajería interno (usualmente un `communication_module` ejecutándose en el mismo nodo/contenedor).

-   **Suscripciones (Outgoing):**
    -   `telemetry_topic`: Mensajes de estado del propio UAV.
    -   `messages_topic`: Mensajes P2P o broadcast generados por algoritmos locales (ej. `mission`).
-   **Publicaciones (Incoming):**
    -   `external_telemetry_topic`: Telemetría recibida de otros UAVs.
    -   `external_messages_topic`: Mensajes recibidos de otros UAVs.

### 2. Simulador de Red Externo (Global)
Se comunica mediante **UDP** con una IP y puerto centralizados donde corre el Simulador de Red de ArduSim2.

## Tipos de Mensajes

Todos los mensajes que viajan hacia/desde el simulador externo utilizan un sobre (envelope) JSON llamado `NetSimMessage`.

### Estructura `NetSimMessage`
```json
{
  "topic": "string",   // "telemetry" o "message"
  "uav_id": "string",  // ID del UAV origen
  "payload": { ... }   // Datos arbitrarios del mensaje (JSON)
}
```

### Flujos de Datos

| Dirección | Tópico Origen | Envoltura Externa | Tópico Destino (Interno) |
| :--- | :--- | :--- | :--- |
| **Salida** | `SubTelemetryTopic` | `"topic": "telemetry"` | Envia a NetSim |
| **Salida** | `SubMessagesTopic` | `"topic": "message"` | Envia a NetSim |
| **Entrada**| Desde NetSim | `"topic": "telemetry"` | `PubExternalTelemetryTopic` |
| **Entrada**| Desde NetSim | `"topic": "message"` | `PubExternalMessagesTopic` |

## Configuración (`config.json`)

El servicio requiere un archivo `config.json` con la siguiente estructura:

```json
{
  "broker_ip": "127.0.0.1",
  "broker_port": 1883,
  "simulator_ip": "172.18.0.1",
  "simulator_port": 5000,
  "sub_telemetry_topic": "uav/1/telemetry",
  "sub_messages_topic": "uav/1/outbox",
  "pub_ext_telemetry_topic": "swarm/telemetry",
  "pub_ext_messages_topic": "swarm/inbox",
  "uav_id": 1
}
```

## Ejecución

El punto de entrada se encuentra en `cmd/main.go`. Utiliza una arquitectura limpia (Hexagonal/Ports & Adapters) para separar la lógica del puente (`usecase`) de los detalles de implementación de red (`infrastructure`).

```bash
go run cmd/main.go
```
