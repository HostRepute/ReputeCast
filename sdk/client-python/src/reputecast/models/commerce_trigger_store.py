from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_collection import CommerceCollection
  from ..models.commerce_product_ref import CommerceProductRef





T = TypeVar("T", bound="CommerceTriggerStore")



@_attrs_define
class CommerceTriggerStore:
    """ 
        Attributes:
            address (str):
            checkouts (bool):
            collections (list[CommerceCollection]):
            currency (str):
            id (str):
            name (str):
            platform (str):
            products (list[CommerceProductRef]):
            status (str):
     """

    address: str
    checkouts: bool
    collections: list[CommerceCollection]
    currency: str
    id: str
    name: str
    platform: str
    products: list[CommerceProductRef]
    status: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_collection import CommerceCollection # noqa: PLC0415
        from ..models.commerce_product_ref import CommerceProductRef # noqa: PLC0415
        address = self.address

        checkouts = self.checkouts

        collections = []
        for collections_item_data in self.collections:
            collections_item = collections_item_data.to_dict()
            collections.append(collections_item)



        currency = self.currency

        id = self.id

        name = self.name

        platform = self.platform

        products = []
        for products_item_data in self.products:
            products_item = products_item_data.to_dict()
            products.append(products_item)



        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "checkouts": checkouts,
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
        from ..models.commerce_product_ref import CommerceProductRef # noqa: PLC0415
        d = dict(src_dict)
        address = d.pop("address")

        checkouts = d.pop("checkouts")

        collections = []
        _collections = d.pop("collections")
        for collections_item_data in (_collections):
            collections_item = CommerceCollection.from_dict(collections_item_data)



            collections.append(collections_item)


        currency = d.pop("currency")

        id = d.pop("id")

        name = d.pop("name")

        platform = d.pop("platform")

        products = []
        _products = d.pop("products")
        for products_item_data in (_products):
            products_item = CommerceProductRef.from_dict(products_item_data)



            products.append(products_item)


        status = d.pop("status")

        commerce_trigger_store = cls(
            address=address,
            checkouts=checkouts,
            collections=collections,
            currency=currency,
            id=id,
            name=name,
            platform=platform,
            products=products,
            status=status,
        )

        return commerce_trigger_store

