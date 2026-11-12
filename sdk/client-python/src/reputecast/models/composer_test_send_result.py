from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ComposerTestSendResult")



@_attrs_define
class ComposerTestSendResult:
    """ 
        Attributes:
            languages (list[str]):
            provider (str):
            sent (list[str]):
     """

    languages: list[str]
    provider: str
    sent: list[str]





    def to_dict(self) -> dict[str, Any]:
        languages = self.languages



        provider = self.provider

        sent = self.sent




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "languages": languages,
            "provider": provider,
            "sent": sent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        languages = cast(list[str], d.pop("languages"))


        provider = d.pop("provider")

        sent = cast(list[str], d.pop("sent"))


        composer_test_send_result = cls(
            languages=languages,
            provider=provider,
            sent=sent,
        )

        return composer_test_send_result

