from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HostReputeAPIKeyInputBody")



@_attrs_define
class HostReputeAPIKeyInputBody:
    """ 
        Attributes:
            api_key (str): A HostRepute API key (hr_live_…) with the monitor:read, alerts:read and assets:write scopes.
                Write-only: it is stored encrypted and never returned.
     """

    api_key: str





    def to_dict(self) -> dict[str, Any]:
        api_key = self.api_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "apiKey": api_key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_key = d.pop("apiKey")

        host_repute_api_key_input_body = cls(
            api_key=api_key,
        )

        return host_repute_api_key_input_body

