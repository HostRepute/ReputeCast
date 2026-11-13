from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="CommerceStockRequestInput")



@_attrs_define
class CommerceStockRequestInput:
    """ 
        Attributes:
            email (str): An existing contact's address
            product_id (str):
            store_id (str | Unset):
     """

    email: str
    product_id: str
    store_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        product_id = self.product_id

        store_id = self.store_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "productId": product_id,
        })
        if store_id is not UNSET:
            field_dict["storeId"] = store_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        product_id = d.pop("productId")

        store_id = d.pop("storeId", UNSET)

        commerce_stock_request_input = cls(
            email=email,
            product_id=product_id,
            store_id=store_id,
        )

        return commerce_stock_request_input

