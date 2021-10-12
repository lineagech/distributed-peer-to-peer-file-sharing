module server

go 1.16

replace connect => ../connect

require (
	connect v0.0.0-00010101000000-000000000000
	messages v0.0.0-00010101000000-000000000000 // indirect
	snapshot v0.0.0-00010101000000-000000000000 // indirect
)

replace messages => ../messages

replace snapshot => ../snapshot
