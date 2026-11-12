from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductOrderConsent")



@_attrs_define
class ProductOrderConsent:
    """ 
        Attributes:
            ip (str):
            list_ (str):
            offered (bool):
            page (str):
            text (str):
            ticked (bool):
            at (datetime.datetime | Unset):
     """

    ip: str
    list_: str
    offered: bool
    page: str
    text: str
    ticked: bool
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        ip = self.ip

        list_ = self.list_

        offered = self.offered

        page = self.page

        text = self.text

        ticked = self.ticked

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "ip": ip,
            "list": list_,
            "offered": offered,
            "page": page,
            "text": text,
            "ticked": ticked,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        ip = d.pop("ip")

        list_ = d.pop("list")

        offered = d.pop("offered")

        page = d.pop("page")

        text = d.pop("text")

        ticked = d.pop("ticked")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        product_order_consent = cls(
            ip=ip,
            list_=list_,
            offered=offered,
            page=page,
            text=text,
            ticked=ticked,
            at=at,
        )

        return product_order_consent

