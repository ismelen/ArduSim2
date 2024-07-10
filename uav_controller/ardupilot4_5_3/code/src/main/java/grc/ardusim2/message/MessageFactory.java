package grc.ardusim2.message;

import grc.ardusim2.Config;
import io.dronefleet.mavlink.MavlinkMessage;
import io.dronefleet.mavlink.common.*;
import io.dronefleet.mavlink.minimal.Heartbeat;

public class MessageFactory {

    public static Message identifyMessage(MavlinkMessage<?> inMsg) {
        Object payload = inMsg.getPayload();
        if (payload instanceof Heartbeat) {
            return new HeartbeatMessage(inMsg);
        } else if (payload instanceof Statustext) {
            return new StatusTextMessage(inMsg);
        } else if (payload instanceof SysStatus) {
            return new SysStatusMessage(inMsg);
        } else if (payload instanceof GlobalPositionInt) {
            return new GlobalPositionIntMessage(inMsg);
        } else if (payload instanceof CommandAck) {
            return new CommandAckMessage(inMsg);
        } else if (payload instanceof GpsStatus) {
            return new GPSStatus(inMsg);
        } else if (payload instanceof AutopilotVersion) {
            return new AutopilotVersionMessage(inMsg);
        } else if (payload instanceof ParamValue){
            return new ParamValueMessage(inMsg);
        } else if (payload instanceof GpsGlobalOrigin){
            return new GPSGlobalOriginMessage(inMsg);
        } else if (payload instanceof HomePosition){
            return new HomePositionMessage(inMsg);
        }else if (payload instanceof Timesync){
            return null;
        } else if (payload instanceof EfiStatus){
            return null;
        } else {
            Config.logger.warn("Received unknown message {}",inMsg);
            return null;
        }
    }
}
