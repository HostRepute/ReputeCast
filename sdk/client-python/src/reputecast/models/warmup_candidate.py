from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="WarmupCandidate")



@_attrs_define
class WarmupCandidate:
    """ 
        Attributes:
            ips (list[str]):
            name (str):
            provider_id (str):
     """

    ips: list[str]
    name: str
    provider_id: str





    def to_dict(self) -> dict[str, Any]:
        ips = self.ips



        name = self.name

        provider_id = self.provider_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "ips": ips,
            "name": name,
            "providerId": provider_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        ips = cast(list[str], d.pop("ips"))


        name = d.pop("name")

        provider_id = d.pop("providerId")

        warmup_candidate = cls(
            ips=ips,
            name=name,
            provider_id=provider_id,
        )

        return warmup_candidate

