package dtos

type IDInput struct {
	ID uint64 `path:"id" minimum:"1"`
}

type PublicRefInput struct {
	PublicRef string `path:"publicRef" format:"uuid"`
}
