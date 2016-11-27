
package handlers

//code generated by 'go generate', do not edit

import (
	"github.com/jargv/plumbus"
	"net/http"
	"reflect"
	"encoding/json"
	"strconv"
	"fmt"
	"log"
)

// avoid unused import errors
var _ json.Delim
var _ log.Logger
var _ fmt.Formatter
var _ strconv.NumError

func init(){
	var dummy func(
		
			*RequestBodyBody,
		
	)(
		
	)

	typ := reflect.TypeOf(dummy)
	plumbus.RegisterAdaptor(typ, func(handler interface{}) http.HandlerFunc {
		callback := handler.(func(
			
				*RequestBodyBody,
			
		)(
			
		))

		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request){
			
			
			
				var arg0 *RequestBodyBody
					if err := json.NewDecoder(req.Body).Decode(&arg0); err != nil {
						msg := fmt.Sprintf("{\"error\": \"decoding json: %s\"}", err.Error())
						http.Error(res, msg, http.StatusBadRequest)
						return
					}
				
			

			
			

			callback(
				
					arg0,
				
			)

			
			

			
		})
	})
}