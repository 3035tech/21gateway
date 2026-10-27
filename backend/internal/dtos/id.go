package dtos

type IDInput struct {
	ID uint64 `path:"id" minimum:"1"`
}
