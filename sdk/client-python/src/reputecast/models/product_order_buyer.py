from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_order_buyer_kind import ProductOrderBuyerKind
from ..models.product_order_buyer_list_status import ProductOrderBuyerListStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductOrderBuyer")



@_attrs_define
class ProductOrderBuyer:
    """ 
        Attributes:
            double_opt_in (bool):
            email (str):
            kind (ProductOrderBuyerKind):
            list_ (str):
            list_status (ProductOrderBuyerListStatus):
            name (str):
            confirmed_at (datetime.datetime | Unset):
            contact_id (str | Unset):
            contact_since (datetime.datetime | Unset):
     """

    double_opt_in: bool
    email: str
    kind: ProductOrderBuyerKind
    list_: str
    list_status: ProductOrderBuyerListStatus
    name: str
    confirmed_at: datetime.datetime | Unset = UNSET
    contact_id: str | Unset = UNSET
    contact_since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        double_opt_in = self.double_opt_in

        email = self.email

        kind = self.kind.value

        list_ = self.list_

        list_status = self.list_status.value

        name = self.name

        confirmed_at: str | Unset = UNSET
        if not isinstance(self.confirmed_at, Unset):
            confirmed_at = self.confirmed_at.isoformat()

        contact_id = self.contact_id

        contact_since: str | Unset = UNSET
        if not isinstance(self.contact_since, Unset):
            contact_since = self.contact_since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "doubleOptIn": double_opt_in,
            "email": email,
            "kind": kind,
            "list": list_,
            "listStatus": list_status,
            "name": name,
        })
        if confirmed_at is not UNSET:
            field_dict["confirmedAt"] = confirmed_at
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if contact_since is not UNSET:
            field_dict["contactSince"] = contact_since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        double_opt_in = d.pop("doubleOptIn")

        email = d.pop("email")

        kind = ProductOrderBuyerKind(d.pop("kind"))




        list_ = d.pop("list")

        list_status = ProductOrderBuyerListStatus(d.pop("listStatus"))




        name = d.pop("name")

        _confirmed_at = d.pop("confirmedAt", UNSET)
        confirmed_at: datetime.datetime | Unset
        if isinstance(_confirmed_at,  Unset):
            confirmed_at = UNSET
        else:
            confirmed_at = datetime.datetime.fromisoformat(_confirmed_at)




        contact_id = d.pop("contactId", UNSET)

        _contact_since = d.pop("contactSince", UNSET)
        contact_since: datetime.datetime | Unset
        if isinstance(_contact_since,  Unset):
            contact_since = UNSET
        else:
            contact_since = datetime.datetime.fromisoformat(_contact_since)




        product_order_buyer = cls(
            double_opt_in=double_opt_in,
            email=email,
            kind=kind,
            list_=list_,
            list_status=list_status,
            name=name,
            confirmed_at=confirmed_at,
            contact_id=contact_id,
            contact_since=contact_since,
        )

        return product_order_buyer

