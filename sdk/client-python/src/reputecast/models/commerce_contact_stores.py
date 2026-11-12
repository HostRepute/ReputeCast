from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.commerce_contact_order import CommerceContactOrder
  from ..models.commerce_contact_store import CommerceContactStore





T = TypeVar("T", bound="CommerceContactStores")



@_attrs_define
class CommerceContactStores:
    """ 
        Attributes:
            orders (list[CommerceContactOrder]):
            stores (list[CommerceContactStore]):
     """

    orders: list[CommerceContactOrder]
    stores: list[CommerceContactStore]





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_contact_order import CommerceContactOrder # noqa: PLC0415
        from ..models.commerce_contact_store import CommerceContactStore # noqa: PLC0415
        orders = []
        for orders_item_data in self.orders:
            orders_item = orders_item_data.to_dict()
            orders.append(orders_item)



        stores = []
        for stores_item_data in self.stores:
            stores_item = stores_item_data.to_dict()
            stores.append(stores_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "orders": orders,
            "stores": stores,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_contact_order import CommerceContactOrder # noqa: PLC0415
        from ..models.commerce_contact_store import CommerceContactStore # noqa: PLC0415
        d = dict(src_dict)
        orders = []
        _orders = d.pop("orders")
        for orders_item_data in (_orders):
            orders_item = CommerceContactOrder.from_dict(orders_item_data)



            orders.append(orders_item)


        stores = []
        _stores = d.pop("stores")
        for stores_item_data in (_stores):
            stores_item = CommerceContactStore.from_dict(stores_item_data)



            stores.append(stores_item)


        commerce_contact_stores = cls(
            orders=orders,
            stores=stores,
        )

        return commerce_contact_stores

