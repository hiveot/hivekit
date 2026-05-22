
// MakeDigiTwinThingID returns the thingID that represents the digital twin Thing
// This is constructed  as: "dtw:{agentID}:{id}"
function makeDigiTwinThingID(agentID: string, thingID: string): string {
    return "dtw:"+agentID+":"+ thingID
}
