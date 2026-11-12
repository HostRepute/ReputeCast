from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.routing_terms_bulk import RoutingTermsBulk
from typing import cast






T = TypeVar("T", bound="RoutingTerms")



@_attrs_define
class RoutingTerms:
    """ 
        Attributes:
            active (bool):
            available (bool):
            bulk (RoutingTermsBulk):
            note (str):
            streams (list[str]):
            variant (str):
     """

    active: bool
    available: bool
    bulk: RoutingTermsBulk
    note: str
    streams: list[str]
    variant: str





    def to_dict(self) -> dict[str, Any]:
        active = self.active

        available = self.available

        bulk = self.bulk.value

        note = self.note

        streams = self.streams



        variant = self.variant


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "available": available,
            "bulk": bulk,
            "note": note,
            "streams": streams,
            "variant": variant,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active = d.pop("active")

        available = d.pop("available")

        bulk = RoutingTermsBulk(d.pop("bulk"))




        note = d.pop("note")

        streams = cast(list[str], d.pop("streams"))


        variant = d.pop("variant")

        routing_terms = cls(
            active=active,
            available=available,
            bulk=bulk,
            note=note,
            streams=streams,
            variant=variant,
        )

        return routing_terms

