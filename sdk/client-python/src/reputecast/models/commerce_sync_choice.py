from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="CommerceSyncChoice")



@_attrs_define
class CommerceSyncChoice:
    """ 
        Attributes:
            checkouts (bool):
            customers (bool):
            orders (bool):
            products (bool):
            write_back (bool):
            list_id (str | Unset):
     """

    checkouts: bool
    customers: bool
    orders: bool
    products: bool
    write_back: bool
    list_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checkouts = self.checkouts

        customers = self.customers

        orders = self.orders

        products = self.products

        write_back = self.write_back

        list_id = self.list_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkouts": checkouts,
            "customers": customers,
            "orders": orders,
            "products": products,
            "writeBack": write_back,
        })
        if list_id is not UNSET:
            field_dict["listId"] = list_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checkouts = d.pop("checkouts")

        customers = d.pop("customers")

        orders = d.pop("orders")

        products = d.pop("products")

        write_back = d.pop("writeBack")

        list_id = d.pop("listId", UNSET)

        commerce_sync_choice = cls(
            checkouts=checkouts,
            customers=customers,
            orders=orders,
            products=products,
            write_back=write_back,
            list_id=list_id,
        )

        return commerce_sync_choice

