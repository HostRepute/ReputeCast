from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AudienceListSummary")



@_attrs_define
class AudienceListSummary:
    """ 
        Attributes:
            created_at (datetime.datetime):
            description (str):
            double_opt_in (bool):
            id (str):
            name (str):
            pending (int):
            subscribed (int):
            origin (str | Unset):
     """

    created_at: datetime.datetime
    description: str
    double_opt_in: bool
    id: str
    name: str
    pending: int
    subscribed: int
    origin: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        description = self.description

        double_opt_in = self.double_opt_in

        id = self.id

        name = self.name

        pending = self.pending

        subscribed = self.subscribed

        origin = self.origin


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "description": description,
            "doubleOptIn": double_opt_in,
            "id": id,
            "name": name,
            "pending": pending,
            "subscribed": subscribed,
        })
        if origin is not UNSET:
            field_dict["origin"] = origin

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        description = d.pop("description")

        double_opt_in = d.pop("doubleOptIn")

        id = d.pop("id")

        name = d.pop("name")

        pending = d.pop("pending")

        subscribed = d.pop("subscribed")

        origin = d.pop("origin", UNSET)

        audience_list_summary = cls(
            created_at=created_at,
            description=description,
            double_opt_in=double_opt_in,
            id=id,
            name=name,
            pending=pending,
            subscribed=subscribed,
            origin=origin,
        )

        return audience_list_summary

