from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HostReputeScope")



@_attrs_define
class HostReputeScope:
    """ 
        Attributes:
            granted (bool):
            key (str):
            use (str):
     """

    granted: bool
    key: str
    use: str





    def to_dict(self) -> dict[str, Any]:
        granted = self.granted

        key = self.key

        use = self.use


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "granted": granted,
            "key": key,
            "use": use,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        granted = d.pop("granted")

        key = d.pop("key")

        use = d.pop("use")

        host_repute_scope = cls(
            granted=granted,
            key=key,
            use=use,
        )

        return host_repute_scope

