from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_contact_store_marketing import CommerceContactStoreMarketing
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceContactStore")



@_attrs_define
class CommerceContactStore:
    """ 
        Attributes:
            currency (str):
            marketing (CommerceContactStoreMarketing):
            name (str):
            on_list (bool):
            orders (int):
            platform (str):
            spent (int):
            store_id (str):
            consent_at (datetime.datetime | Unset):
            list_name (str | Unset):
     """

    currency: str
    marketing: CommerceContactStoreMarketing
    name: str
    on_list: bool
    orders: int
    platform: str
    spent: int
    store_id: str
    consent_at: datetime.datetime | Unset = UNSET
    list_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        marketing = self.marketing.value

        name = self.name

        on_list = self.on_list

        orders = self.orders

        platform = self.platform

        spent = self.spent

        store_id = self.store_id

        consent_at: str | Unset = UNSET
        if not isinstance(self.consent_at, Unset):
            consent_at = self.consent_at.isoformat()

        list_name = self.list_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "marketing": marketing,
            "name": name,
            "onList": on_list,
            "orders": orders,
            "platform": platform,
            "spent": spent,
            "storeId": store_id,
        })
        if consent_at is not UNSET:
            field_dict["consentAt"] = consent_at
        if list_name is not UNSET:
            field_dict["listName"] = list_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency")

        marketing = CommerceContactStoreMarketing(d.pop("marketing"))




        name = d.pop("name")

        on_list = d.pop("onList")

        orders = d.pop("orders")

        platform = d.pop("platform")

        spent = d.pop("spent")

        store_id = d.pop("storeId")

        _consent_at = d.pop("consentAt", UNSET)
        consent_at: datetime.datetime | Unset
        if isinstance(_consent_at,  Unset):
            consent_at = UNSET
        else:
            consent_at = datetime.datetime.fromisoformat(_consent_at)




        list_name = d.pop("listName", UNSET)

        commerce_contact_store = cls(
            currency=currency,
            marketing=marketing,
            name=name,
            on_list=on_list,
            orders=orders,
            platform=platform,
            spent=spent,
            store_id=store_id,
            consent_at=consent_at,
            list_name=list_name,
        )

        return commerce_contact_store

