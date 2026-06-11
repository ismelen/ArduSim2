# Controllers

This directory contains the UAV Controllers available for the simulation.

## How to add a new UAV Controller

To add a new UAV Controller that is compatible with the GUI and the dynamic build system, follow these steps:

1. **Create a Folder**: Create a new folder inside `src/controllers/` with the name of your controller (e.g., `my_custom_controller`).

2. **Add a `schema.json`**: This file is required for the GUI to discover and configure your controller. It must contain at least `service_id` and `title`.
   ```json
   {
       "service_id": "my_custom_controller",
       "title": "My Custom Controller",
       "properties": {
           // Any configuration properties your controller requires
       }
   }
   ```

3. **Add a Base Dockerfile**: Your controller folder must include a `Dockerfile` (or `SITL` file) that defines the **base image** for your controller. 
   - *Important*: This Dockerfile should NOT copy the ArduCopter binary directly. It should only install dependencies, copy your controller's source code, and set up the execution environment (e.g., `run.sh`).
   - The simulation build system will automatically create a derived image that inherits from your base image and copies the user-selected ArduCopter binary into `/app/arducopter`.

4. **Execution Script**: Ensure your entrypoint (e.g., `run.sh`) expects the ArduCopter binary to be located at `/app/arducopter` and has execution permissions.

5. **Configuration**: If your controller requires a `config.json`, include it in your folder. The GUI will generate a specific config file for each UAV and mount it to `/app/config.json`.

By following these steps, your controller will be automatically discovered by the GUI, and users will be able to select it for their UAVs along with any ArduCopter binary.
