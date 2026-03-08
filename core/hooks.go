package core

type Hooks struct {
	OnLogin    func(user *User)
	OnRegister func(user *User)
}
