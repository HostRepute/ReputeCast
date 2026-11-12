from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SmsSenderOption")



@_attrs_define
class SmsSenderOption:
    """ 
        Attributes:
            adapter (str):
            countries (list[str]):
            id (str):
            kind (str):
            label (str):
            provider (str):
            sendable (list[str]):
     """

    adapter: str
    countries: list[str]
    id: str
    kind: str
    label: str
    provider: str
    sendable: list[str]





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        countries = self.countries



        id = self.id

        kind = self.kind

        label = self.label

        provider = self.provider

        sendable = self.sendable




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "countries": countries,
            "id": id,
            "kind": kind,
            "label": label,
            "provider": provider,
            "sendable": sendable,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        countries = cast(list[str], d.pop("countries"))


        id = d.pop("id")

        kind = d.pop("kind")

        label = d.pop("label")

        provider = d.pop("provider")

        sendable = cast(list[str], d.pop("sendable"))


        sms_sender_option = cls(
            adapter=adapter,
            countries=countries,
            id=id,
            kind=kind,
            label=label,
            provider=provider,
            sendable=sendable,
        )

        return sms_sender_option

