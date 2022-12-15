package customer

import (
	"config-service/db"
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func AddRoutes(g *gin.Engine) {
	customer := g.Group("/")

	customer.Use(handlers.DBContextMiddleware(consts.CustomersCollection))

	customer.GET("customer", getCustomer)
	customer.POST("customer_tenant", postCustomerTenant)
}

func getCustomer(c *gin.Context) {
	defer log.LogNTraceEnterExit("getCustomer", c)()
	_, customerGUID, err := db.ReadContext(c)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to read customer guid from context", err)
		return
	}
	if doc, err := db.GetDocByGUID[*types.Customer](c, customerGUID); err != nil {
		handlers.ResponseInternalServerError(c, "failed to read document", err)
		return
	} else if doc == nil {
		handlers.ResponseDocumentNotFound(c)
		return
	} else {
		c.JSON(http.StatusOK, doc)
	}
}

func postCustomerTenant(c *gin.Context) {
	defer log.LogNTraceEnterExit("postCustomerTenant", c)()
	var customer *types.Customer
	if err := c.ShouldBindBodyWith(&customer, binding.JSON); err != nil || customer == nil {
		handlers.ResponseFailedToBindJson(c, err)
		return
	}
	if customer.GUID == "" {
		handlers.ResponseMissingGUID(c)
		return
	}
	customer.InitNew()
	dbDoc := types.Document[*types.Customer]{
		ID:        customer.GUID,
		Content:   customer,
		Customers: []string{customer.GUID},
	}
	handlers.PostDBDocumentHandler(c, dbDoc)
}