from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_i_ps_input_mode import ProviderIPsInputMode
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ProviderIPsInput")



@_attrs_define
class ProviderIPsInput:
    """ 
        Attributes:
            mode (ProviderIPsInputMode):
            ips (list[str] | Unset): The public IPv4 or IPv6 addresses mail leaves from, up to 64, when mode is dedicated
     """

    mode: ProviderIPsInputMode
    ips: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        ips: list[str] | Unset = UNSET
        if not isinstance(self.ips, Unset):
            ips = self.ips




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })
        if ips is not UNSET:
            field_dict["ips"] = ips

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = ProviderIPsInputMode(d.pop("mode"))




        ips = cast(list[str], d.pop("ips", UNSET))


        provider_i_ps_input = cls(
            mode=mode,
            ips=ips,
        )

        return provider_i_ps_input

