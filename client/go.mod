module client

go 1.16

replace connect => ../connect

replace messages => ../messages

require (
	connect v0.0.0-00010101000000-000000000000
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac
	messages v0.0.0-00010101000000-000000000000
	snapshot v0.0.0-00010101000000-000000000000
)

replace snapshot => ../snapshot
