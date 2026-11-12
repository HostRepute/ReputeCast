from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PlacementEvent")



@_attrs_define
class PlacementEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            type_ (str): delivered, deferred, soft_bounce, hard_bounce, complaint, rejected…
            code (str | Unset):
            diagnostic (str | Unset):
            host (str | Unset):
     """

    at: datetime.datetime
    type_: str
    code: str | Unset = UNSET
    diagnostic: str | Unset = UNSET
    host: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        type_ = self.type_

        code = self.code

        diagnostic = self.diagnostic

        host = self.host


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "type": type_,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if diagnostic is not UNSET:
            field_dict["diagnostic"] = diagnostic
        if host is not UNSET:
            field_dict["host"] = host

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        type_ = d.pop("type")

        code = d.pop("code", UNSET)

        diagnostic = d.pop("diagnostic", UNSET)

        host = d.pop("host", UNSET)

        placement_event = cls(
            at=at,
            type_=type_,
            code=code,
            diagnostic=diagnostic,
            host=host,
        )

        return placement_event

