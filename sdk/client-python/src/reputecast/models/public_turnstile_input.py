from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PublicTurnstileInput")



@_attrs_define
class PublicTurnstileInput:
    """ 
        Attributes:
            site_key (str):  Example: 0x4AAAAAAA….
            secret (str | Unset): Write-only; omit to keep the stored secret
     """

    site_key: str
    secret: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        site_key = self.site_key

        secret = self.secret


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "siteKey": site_key,
        })
        if secret is not UNSET:
            field_dict["secret"] = secret

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        site_key = d.pop("siteKey")

        secret = d.pop("secret", UNSET)

        public_turnstile_input = cls(
            site_key=site_key,
            secret=secret,
        )

        return public_turnstile_input

