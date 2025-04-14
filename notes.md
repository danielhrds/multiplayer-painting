# TO DO

NewPlayer default params (if golang allow)
BasePlayer type composition
Bezier curves for smooth lines
Colors
Redo/Undo

# Events

There's an event queue accumulated within a tick and processed after tick.

### Kind
- JOINED
- LEFT
- DRAWING
- DRAW
- DONE

# How it should work

When a client presses the mousedown button, it sends an **DRAWING** event,
then the server updates client's state to drawing equal to true and returns
an **DRAWING** event to all active users assuring that the client is in fact drawing.
That way the server knows if it should split the pixels into a new array. 
On client's side is created a new array every time
it gets a **DRAWING** event, all users need to do that. Right after the **DRAWING**
event is sent, **DRAW** event is sent too so the server can start appending updates
to the update buffer. After each tick, it sends the current state of the update
buffer, update the main map appending the new pixels and frees the update buffer.

When a client releases mousedown button, it sends an **DONE** event and the server
updates client state to drawing equal to false for all users.


# Server

### Update buffer

Store the updates sent by the client. Might store the whole package.

## Data

Stores an id and a client struct in a map to keep track
of the client's state. The pixels field in client struct
consist in an array of array of pixels. That way the server
can undo and redo drawings of each client.


**EVENTS RECEIVED BY THE SERVER**
- JOINED:  Add client to clients map. Send all clients' pixels to reconstruct the board.
- LEFT:    Remove client from clients map
- DRAWING: Update client state to drawing equal to true and notify all active users.
- DRAW:    Accumulate pixels on a pixel buffer.
           _OBS:_ It frees the update buffer after each tick
           _OBS2:_ Need to free the undo array.
- DONE:    Update all users that were drawing to false.
           _OBS:_ Check if last array is empty, if so, remove that.

Incomplete

- UNDO:   Remove last drawing. Notify all users.
          _OBS:_ Add to a dedicated map to keep track of a stack of undo.
- REDO:   Add last removed draw. Notify all active users with the whole drawing.


**EVENTS SENT BY THE SERVER**
- DRAWING: Notify all users that are drawing
- DRAW:    Send all accumulated pixels.
- DONE:    Notify all users that are done drawing.

# Client

## Data

Stores an id and an array of array of pixels in a map. The only thing clients do
with it is iterate over the map and over each array to draw each pixel.
_OBS:_ Clients' draws are independent, so it's possible to draw them in parallel.

**EVENTS RECEIVED BY THE CLIENT**
- JOINED:  Add player to the players map
- DRAWING: Update all users drawing to true and create a new array of pixels.
- DRAW:    Add corresponding pixels to correct key on clients map.
           data i.e.: map[int]*[]*Pixel
           _change int to **type id = int**_
- DONE:    Update all users that were drawing to false.
           _OBS:_ Check if last array is empty, if so, remove that.

**EVENTS SENT BY THE CLIENT**
- JOINED:  Notify server that you joined.
- LEFT:    Notify server that you left.
- DRAWING: Notify server that you are drawing.
- DRAW:    Send pixels in real-time with an id so the server
           can know when to split into a new array of pixels
- DONE:    Notify server that you are no long drawing.
