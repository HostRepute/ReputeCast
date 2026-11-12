from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="CommerceStockRequest")



@_attrs_define
class CommerceStockRequest:
    """ 
        Attributes:
            available (bool): The product is in stock now: back-in-stock flows start on its next return
            created_at (datetime.datetime):
            product_id (str): ReputeCast's id of the product
            store_id (str):
            title (str):  Example: Deploy Friday cap.
     """

    available: bool
    created_at: datetime.datetime
    product_id: str
    store_id: str
    title: str





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        created_at = self.created_at.isoformat()

        product_id = self.product_id

        store_id = self.store_id

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "createdAt": created_at,
            "productId": product_id,
            "storeId": store_id,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        product_id = d.pop("productId")

        store_id = d.pop("storeId")

        title = d.pop("title")

        commerce_stock_request = cls(
            available=available,
            created_at=created_at,
            product_id=product_id,
            store_id=store_id,
            title=title,
        )

        return commerce_stock_request

