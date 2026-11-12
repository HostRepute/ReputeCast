from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="HostReputeShared")



@_attrs_define
class HostReputeShared:
    """ 
        Attributes:
            domains (list[str]):
            ips (list[str]):
     """

    domains: list[str]
    ips: list[str]





    def to_dict(self) -> dict[str, Any]:
        domains = self.domains



        ips = self.ips




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domains": domains,
            "ips": ips,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domains = cast(list[str], d.pop("domains"))


        ips = cast(list[str], d.pop("ips"))


        host_repute_shared = cls(
            domains=domains,
            ips=ips,
        )

        return host_repute_shared

