module client

go 1.16

replace connect => ../connect

replace messages => ../messages

require (
	connect v0.0.0-00010101000000-000000000000
	messages v0.0.0-00010101000000-000000000000 // indirect
)
