package grc.ardusim2.command;

import grc.ardusim2.APIThread;
import grc.ardusim2.Config;
import grc.ardusim2.drone.Drone;
import grc.ardusim2.drone.FlightMode;
import grc.ardusim2.drone.FlightModes;
import io.dronefleet.mavlink.common.CommandLong;
import io.dronefleet.mavlink.common.MavCmd;
import org.json.JSONObject;

import java.util.function.Function;

public class SetFlightmode extends Command {

    private final Drone drone = Drone.getInstance();
    private final APIThread api = APIThread.getInstance();

    private final FlightModes mode;
    private final static String FLIGHTMODE = "flightmode";

    public SetFlightmode(FlightModes mode) {
        super();
        super.commandID = MavCmd.MAV_CMD_DO_SET_MODE.ordinal();
        this.mode = mode;
        super.payload = CommandLong.builder()
                .targetSystem(Drone.getInstance().getMavID())
                .targetComponent(0) // MavComponent.MAV_COMP_ID_ALL
                .command(MavCmd.MAV_CMD_DO_SET_MODE)
                .param1(1)
                .param2(mode.ordinal())
                .build();
    }

    public SetFlightmode(JSONObject APIrequest) {
        this(FlightModes.valueOf(APIrequest.getString(FLIGHTMODE)));
    }

    @Override
    public void processACK() {
        drone.setStatus(Drone.Status.OK);
        Config.logger.info("New flightmode: {}", mode.toString());
        api.respond(this,"ACK");
    }

    @Override
    public void processError() {
        drone.setStatus(Drone.Status.OK);
        Config.logger.warn("Could not set flightmode to {}",mode.toString());
        api.respond(this,"NACK");
    }

    public static Function<JSONObject, Boolean> isValidMessage() {
        return request -> {
            try {
                FlightModes.valueOf(request.getString(FLIGHTMODE));
            }catch (IllegalArgumentException e) {
                Config.logger.warn("Trying to change flightmode but flightmode was not specified");
                return false;
            }
            return true;
        };
    }
}
