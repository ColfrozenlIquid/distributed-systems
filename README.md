# Supported Sensors & Actors

## 🛰️ Sensors

### 1. Drone
- **Sensor:** `SENSOR_TYPE_PERSON_CONFIDENCE`
- **Movement:** `MOVABLE_TYPE_AIR`

### 2. Vibration Sensor
- **Sensor:** `SENSOR_TYPE_VIBRATION`
- **Movement:** `NONE`

### 3. Water Level Sensor
- **Sensor:** `SENSOR_TYPE_WATER_LEVEL`
- **Movement:** `NONE`

---

## 🤖 Actors

### 1. Rescue Vehicle
- **Mission:** `MISSION_TYPE_RESCUE_PERSONS`
- **Movement:** `MOVABLE_TYPE_LAND`

### 2. Vibration Vehicle
- **Mission:** `MISSION_TYPE_STABILIZE_VIBRATION`
- **Movement:** `MOVABLE_TYPE_LAND`

### 3. Water Vehicle
- **Mission:** `MISSION_TYPE_PUMP_WATER`
- **Movement:** `MOVABLE_TYPE_WATER`

# MQTT Topic structure

| **Topic** | Body | Retain? | QoS |
| --- | --- | --- | --- |
| **`tiles/{x}/{y}/readings/{sensor_type}`** | `SensorReading` | ✗ | 0 |
| **`actors/{actor_id}/status`** | `ActorStatus` | ✓ | 1 |
| **`actors/{actor_id}/telemetry`** | `ActorTelemetry` | ✗ | 0 |
| **`missions/{mission_id}/updates`** | `MissionUpdate` | ✗ | 1 |