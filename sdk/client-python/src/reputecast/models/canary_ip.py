from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CanaryIP")



@_attrs_define
class CanaryIP:
    """ 
        Attributes:
            ip (str):
            provider (str):
     """

    ip: str
    provider: str





    def to_dict(self) -> dict[str, Any]:
        ip = self.ip

        provider = self.provider


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "ip": ip,
            "provider": provider,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        ip = d.pop("ip")

        provider = d.pop("provider")

        canary_ip = cls(
            ip=ip,
            provider=provider,
        )

        return canary_ip

