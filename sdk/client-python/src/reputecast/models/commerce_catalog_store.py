from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_collection import CommerceCollection





T = TypeVar("T", bound="CommerceCatalogStore")



@_attrs_define
class CommerceCatalogStore:
    """ 
        Attributes:
            collections (list[CommerceCollection]):
            currency (str):
            id (str):
            name (str):
            platform (str):
            products (int):
            status (str):
     """

    collections: list[CommerceCollection]
    currency: str
    id: str
    name: str
    platform: str
    products: int
    status: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_collection import CommerceCollection # noqa: PLC0415
        collections = []
        for collections_item_data in self.collections:
            collections_item = collections_item_data.to_dict()
            collections.append(collections_item)



        currency = self.currency

        id = self.id

        name = self.name

        platform = self.platform

        products = self.products

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "collections": collections,
            "currency": currency,
            "id": id,
            "name": name,
            "platform": platform,
            "products": products,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_collection import CommerceCollection # noqa: PLC0415
        d = dict(src_dict)
        collections = []
        _collections = d.pop("collections")
        for collections_item_data in (_collections):
            collections_item = CommerceCollection.from_dict(collections_item_data)



            collections.append(collections_item)


        currency = d.pop("currency")

        id = d.pop("id")

        name = d.pop("name")

        platform = d.pop("platform")

        products = d.pop("products")

        status = d.pop("status")

        commerce_catalog_store = cls(
            collections=collections,
            currency=currency,
            id=id,
            name=name,
            platform=platform,
            products=products,
            status=status,
        )

        return commerce_catalog_store

