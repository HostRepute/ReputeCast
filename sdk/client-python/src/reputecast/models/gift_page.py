from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="GiftPage")



@_attrs_define
class GiftPage:
    """ 
        Attributes:
            host (str):
            link_months (int):
            on (bool):
            url (str):
            changed_at (datetime.datetime | Unset):
            changed_by (str | Unset):
     """

    host: str
    link_months: int
    on: bool
    url: str
    changed_at: datetime.datetime | Unset = UNSET
    changed_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        link_months = self.link_months

        on = self.on

        url = self.url

        changed_at: str | Unset = UNSET
        if not isinstance(self.changed_at, Unset):
            changed_at = self.changed_at.isoformat()

        changed_by = self.changed_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "linkMonths": link_months,
            "on": on,
            "url": url,
        })
        if changed_at is not UNSET:
            field_dict["changedAt"] = changed_at
        if changed_by is not UNSET:
            field_dict["changedBy"] = changed_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        link_months = d.pop("linkMonths")

        on = d.pop("on")

        url = d.pop("url")

        _changed_at = d.pop("changedAt", UNSET)
        changed_at: datetime.datetime | Unset
        if isinstance(_changed_at,  Unset):
            changed_at = UNSET
        else:
            changed_at = datetime.datetime.fromisoformat(_changed_at)




        changed_by = d.pop("changedBy", UNSET)

        gift_page = cls(
            host=host,
            link_months=link_months,
            on=on,
            url=url,
            changed_at=changed_at,
            changed_by=changed_by,
        )

        return gift_page

