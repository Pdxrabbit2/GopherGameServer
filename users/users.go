//
package users

import (
	"errors"
	"github.com/hewiefreeman/GopherGameServer/rooms"
	"github.com/hewiefreeman/GopherGameServer/helpers"
	"github.com/gorilla/websocket"
)

//
type User struct {
	name string
	databaseID int64
	isGuest bool

	room string

	status int

	socket *websocket.Conn
}

var (
	userCount int64 = 0
	users map[string]*User = make(map[string]*User)
	usersActionChan *helpers.ActionChannel = helpers.NewActionChannel()
	serverStarted bool = false;
	kickOnLogin bool = false;
)

//USER STATUS LIST
const (
	StatusAvailable = iota // User is available
	StatusInGame // User is in a game
	StatusIdle // User is idle
)

//SEVER START-UP FUNCTIONS
func SetServerStarted(val bool){
	if(!serverStarted){
		serverStarted = val;
	}
}

func SettingsSet(kickDups bool){
	if(!serverStarted){
		kickOnLogin = kickDups;
	}
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//   LOG A USER IN   /////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////

func Login(userName string, dbID int64, isGuest bool, socket *websocket.Conn) (User, error) {
	var err error = nil;

	//REJECT INCORRECT INPUT
	if(len(userName) == 0){
		return User{}, errors.New("users.Login() requires a user name");
	}else if(dbID < -1){
		return User{}, errors.New("users.Login() requires a database ID (or -1 for no ID)");
	}else if(socket == nil){
		return User{}, errors.New("users.Login() requires a socket");
	}

	//ALWAYS SET A GUEST'S id TO -1
	databaseID := dbID
	if isGuest { databaseID = -1 }

	response := usersActionChan.Execute(loginUser, []interface{}{userName, databaseID, isGuest, socket});
	if(response[1] != nil){
		if(){

		}else{
			err = response[1].(error);
		}
	}

	//
	return response[0].(User), err;
}

func loginUser(p []interface{}) []interface{} {
	userName, dbID, isGuest, socket := p[0].(string), p[1].(int64), p[2].(bool), p[3].(*websocket.Conn);
	var userRef User = User{};
	var err error = nil;

	if _, ok := users[userName]; ok {
		err = errors.New("User '"+userName+"' is already logged in");
	}else{
		userCount++;
		newUser := User{name: userName, databaseID: dbID, isGuest: isGuest, socket: socket};
		users[userName] = &newUser;
		userRef = *users[userName];
	}

	return []interface{}{userRef, err};
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//   GET A USER   ////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////

func Get(userName string) (User, error) {
	var err error = nil;

	//REJECT INCORRECT INPUT
	if(len(userName) == 0){ return User{}, errors.New("users.Get() requires a user name"); }

	response := usersActionChan.Execute(getUser, []interface{}{userName});
	if(response[1] != nil){ err = response[1].(error); }

	//
	return response[0].(User), err;
}

func getUser(p []interface{}) []interface{} {
	userName := p[0].(string);
	var userRef User = User{};
	var err error = nil;

	if user, ok := users[userName]; ok {
		userRef = *user;
	}else{
		err = errors.New("User '"+userName+"' is not logged in");
	}

	return []interface{}{userRef, err};
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//   CONVERT A RoomUser INTO A User   ////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////

// Converts a RoomUser into a User.
func RoomUser(ru rooms.RoomUser) (User, error) {
	u, e := Get(ru.Name());
	if(e != nil){ return User{}, e; }
	return u, e;
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//   MAKE A USER JOIN A ROOM   ///////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////

//
func (u *User) Join(r rooms.Room) error {
	if(u.room == r.Name()){
		return errors.New("User '"+u.name+"' is already in room '"+r.Name()+"'");
	}else if(u.room != ""){
		//LEAVE USER'S CURRENT ROOM
		u.Leave();
	}

	//CHANGE User's ROOM NAME
	response := usersActionChan.Execute(changeUserRoomName, []interface{}{u, r.Name()});
	if(response[0] != nil){ return response[0].(error); }

	//ADD USER TO DESIGNATED ROOM
	addErr := r.AddUser(&u.name, u.socket);
	return addErr;
}

func (u *User) Leave() error {
	if(u.room != ""){
		room, roomErr := rooms.Get(u.room);
		if roomErr != nil { return roomErr; }
		//
		removeErr := room.RemoveUser(u.name);
		if(removeErr != nil){ return removeErr; }
	}else{
		return errors.New("User '"+u.name+"' is not in a room.");
	}

	//CHANGE User's ROOM NAME
	response := usersActionChan.Execute(changeUserRoomName, []interface{}{u, ""});
	if(response[0] != nil){ return response[0].(error) }

	return nil;
}

func changeUserRoomName(p []interface{}) []interface{} {
	theUser, roomName := p[0].(*User), p[1].(string);
	var err error = nil;

	if _, ok := users[(*theUser).name]; ok {
		(*users[(*theUser).name]).room = roomName;
		(*theUser).room = roomName
	}else{
		err = errors.New("User '"+theUser.name+"' is not logged in");
	}

	//
	return []interface{}{err}
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//   LOG A USER OUT   ////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////

func (u *User) LogOut() {
	//REMOVE USER FROM THEIR ROOM
	if(u.room != ""){
		room, err := rooms.Get(u.room);
		if(err == nil){
			room.RemoveUser(u.name);
		}
	}
	//LOG USER OUT
	usersActionChan.Execute(logUserOut, []interface{}{u.name});
}

func logUserOut(p []interface{}) []interface{} {
	userName := p[0].(string);
	delete(users, userName);
	return []interface{}{};
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//   KICK A USER   ///////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////

func Kick(userName string) error {
	if(len(userName) == 0){
		return errors.New("users.Kick() requires a user name");
	}
	//
	user, err := Get(userName);
	if(err != nil){
		return err;
	}
	//
	user.LogOut();
	//
	return nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//   User ATTRIBUTE READERS   ////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////

func (u *User) Name() string {
	return u.name;
}

func (u *User) DatabaseID() int64 {
	return u.databaseID;
}

func (u *User) RoomName() string {
	return u.room;
}

func (u *User) Socket() *websocket.Conn {
	return u.socket;
}

func (u *User) IsGuest() bool {
	return u.isGuest;
}
