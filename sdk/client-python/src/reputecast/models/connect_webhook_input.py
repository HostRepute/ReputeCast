from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectWebhookInput")



@_attrs_define
class ConnectWebhookInput:
    """ 
        Attributes:
            public_url (str | Unset):
     """

    public_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        public_url = self.public_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if public_url is not UNSET:
            field_dict["publicUrl"] = public_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        public_url = d.pop("publicUrl", UNSET)

        connect_webhook_input = cls(
            public_url=public_url,
        )

        return connect_webhook_input

