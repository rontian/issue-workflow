package app

import (
	"os"
	"path/filepath"
)

func projectRequireReview(root string)(bool,error){
	raw,err:=os.ReadFile(filepath.Join(root,"iw.json"));if os.IsNotExist(err){return false,nil};if err!=nil{return false,err};cfg,err:=parseProjectConfig(raw);if err!=nil{return false,err};return cfg.Policy.RequireReview,nil
}
