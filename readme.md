To run the program you must first start the server by using the command $ go run server/server.go
From here on you add users to the chat by running a client followed by the username e.g. $ go run client/client.go ‘David’
To write messages to other clients, simply type the message into the terminal and it will be broadcasted to the other clients. Ps no quotes (‘ ‘) are needed here, write the message “raw”.
Lastly for a user to leave the on-going chat simply direct to the given users terminal and press ctrl+c. This will notify the other users that the given person has left (as shown on the third last line on the system log.)
