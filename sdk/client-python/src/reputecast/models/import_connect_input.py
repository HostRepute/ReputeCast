from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ImportConnectInput")



@_attrs_define
class ImportConnectInput:
    """ 
        Attributes:
            api_key (str): Write-only: stored encrypted, never returned
            base_url (str | Unset): listmonk, Keila or Notifuse server address
            user (str | Unset): listmonk API user
     """

    api_key: str
    base_url: str | Unset = UNSET
    user: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        api_key = self.api_key

        base_url = self.base_url

        user = self.user


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "apiKey": api_key,
        })
        if base_url is not UNSET:
            field_dict["baseUrl"] = base_url
        if user is not UNSET:
            field_dict["user"] = user

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_key = d.pop("apiKey")

        base_url = d.pop("baseUrl", UNSET)

        user = d.pop("user", UNSET)

        import_connect_input = cls(
            api_key=api_key,
            base_url=base_url,
            user=user,
        )

        return import_connect_input

