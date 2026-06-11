package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bank-data-db/proto/categories_pb"
	"github.com/bank-data-db/proto/mappings_pb"
	"github.com/bank-data-db/proto/utils"
	"github.com/bank-data-db/server/config"
	"github.com/bank-data-db/server/db"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/grpc/bank_data"
)

func main() {
	cleanup := config.LoadForCLI(true)
	defer cleanup()

	if len(os.Args) < 3 {
		panic("Usage: clone_to [source username] [target username]")
	}

	db := db.GetDB(slog.Default())
	s := store.NewStore(db)

	userSrc, err := s.UserByName(context.Background(), os.Args[1])
	if err != nil {
		panic("Source user: " + err.Error())
	} else if userSrc == nil {
		panic("Source user: not found")
	}

	userDst, err := s.UserByName(context.Background(), os.Args[2])
	if err != nil {
		panic("Target user: " + err.Error())
	} else if userDst == nil {
		panic("Target user: not found")
	}

	ctxSrc := bank_data.ContextWithUserID(context.Background(), userSrc.ID)
	ctxDst := bank_data.ContextWithUserID(context.Background(), userDst.ID)

	api := bank_data.NewAPI(db, s)
	dstMappings, err := utils.ListAll(func(tok *string) (utils.GenericRespList[*mappings_pb.Mapping], error) {
		return api.MappingsList(ctxDst, mappings_pb.ReqList_builder{
			PaginationToken: tok,
		}.Build())
	})
	if err != nil {
		panic("Failed to fetch target user's mappings: " + err.Error())
	}
	dstCategories, err := utils.ListAll(func(tok *string) (utils.GenericRespList[*categories_pb.Category], error) {
		return api.CategoriesList(ctxDst, categories_pb.ReqList_builder{
			PaginationToken: tok,
		}.Build())
	})
	if err != nil {
		panic("Failed to fetch target user's categories: " + err.Error())
	}

	srcMappings, err := utils.ListAll(func(tok *string) (utils.GenericRespList[*mappings_pb.Mapping], error) {
		return api.MappingsList(ctxSrc, mappings_pb.ReqList_builder{
			PaginationToken: tok,
		}.Build())
	})
	if err != nil {
		panic("Failed to fetch source user's mappings: " + err.Error())
	}

	srcCategories, err := utils.ListAll(func(tok *string) (utils.GenericRespList[*categories_pb.Category], error) {
		return api.CategoriesList(ctxSrc, categories_pb.ReqList_builder{
			PaginationToken: tok,
		}.Build())
	})
	if err != nil {
		panic("Failed to fetch source user's categories: " + err.Error())
	}

	for _, src := range srcCategories {
		dupe := false
		for _, v := range dstCategories {
			if src.GetName() == v.GetName() {
				dupe = true
				break
			}
		}

		if dupe {
			fmt.Printf("Category '%v': Skipping due to duplicate\n", src.GetName())
			continue
		}

		reqNew := &categories_pb.ReqNew{}
		utils.CopyTo(reqNew.ProtoReflect(), src.ProtoReflect(), false)

		resp, err := api.CategoriesNew(ctxDst, reqNew)
		if err != nil {
			fmt.Printf("Category '%v': Failed to create: %v\n", src.GetName(), err)
			continue
		}

		fmt.Printf("Category '%v': SUCCESS! (%v)\n", src.GetName(), resp)
		src.SetID(resp.GetID())

		dstCategories = append(dstCategories, src)
	}

	for _, src := range srcMappings {
		dupe := false
		for _, v := range dstMappings {
			if src.GetName() == v.GetName() {
				dupe = true
				break
			}
		}

		if dupe {
			fmt.Printf("Mapping '%v': Skipping due to duplicate\n", src.GetName())
			continue
		}

		reqNew := &mappings_pb.ReqNew{}
		utils.CopyTo(reqNew.ProtoReflect(), src.ProtoReflect(), false)
		if reqNew.HasResultCategoryID() {
			srcCatName := ""
			for _, v := range srcCategories {
				if v.GetID() == reqNew.GetResultCategoryID() {
					srcCatName = v.GetName()
					break
				}
			}
			newID := ""
			for _, v := range dstCategories {
				if srcCatName == v.GetName() {
					newID = v.GetID()
					break
				}
			}
			if newID == "" {
				fmt.Printf("Mapping '%v': Skipped due to unknown category '%v'\n", src.GetName(), srcCatName)
				continue
			}
	
			reqNew.SetResultCategoryID(newID)
		}

		resp, err := api.MappingsNew(ctxDst, reqNew)
		if err != nil {
			fmt.Printf("Mapping '%v': Failed to create: %v\n", src.GetName(), err)
			continue
		}
		fmt.Printf("Mapping '%v': SUCCESS! (%v)\n", src.GetName(), resp)
	}
}
