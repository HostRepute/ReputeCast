from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_send_input_over_limit import WhatsAppSendInputOverLimit
from ..types import UNSET, Unset






T = TypeVar("T", bound="WhatsAppSendInput")



@_attrs_define
class WhatsAppSendInput:
    """ 
        Attributes:
            version (int):
            over_limit (WhatsAppSendInputOverLimit | Unset):
     """

    version: int
    over_limit: WhatsAppSendInputOverLimit | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        over_limit: str | Unset = UNSET
        if not isinstance(self.over_limit, Unset):
            over_limit = self.over_limit.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
        })
        if over_limit is not UNSET:
            field_dict["overLimit"] = over_limit

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        _over_limit = d.pop("overLimit", UNSET)
        over_limit: WhatsAppSendInputOverLimit | Unset
        if isinstance(_over_limit,  Unset):
            over_limit = UNSET
        else:
            over_limit = WhatsAppSendInputOverLimit(_over_limit)




        whats_app_send_input = cls(
            version=version,
            over_limit=over_limit,
        )

        return whats_app_send_input

