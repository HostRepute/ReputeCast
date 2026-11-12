from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_ab_variant_key import ComposerABVariantKey
from ..types import UNSET, Unset






T = TypeVar("T", bound="ComposerABVariant")



@_attrs_define
class ComposerABVariant:
    """ 
        Attributes:
            key (ComposerABVariantKey): Variant A is the campaign itself
            html (str | Unset):
            mjml (str | Unset):
            preheader (str | Unset):
            send_time (str | Unset):
            subject (str | Unset):
     """

    key: ComposerABVariantKey
    html: str | Unset = UNSET
    mjml: str | Unset = UNSET
    preheader: str | Unset = UNSET
    send_time: str | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        html = self.html

        mjml = self.mjml

        preheader = self.preheader

        send_time = self.send_time

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
        })
        if html is not UNSET:
            field_dict["html"] = html
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if send_time is not UNSET:
            field_dict["sendTime"] = send_time
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = ComposerABVariantKey(d.pop("key"))




        html = d.pop("html", UNSET)

        mjml = d.pop("mjml", UNSET)

        preheader = d.pop("preheader", UNSET)

        send_time = d.pop("sendTime", UNSET)

        subject = d.pop("subject", UNSET)

        composer_ab_variant = cls(
            key=key,
            html=html,
            mjml=mjml,
            preheader=preheader,
            send_time=send_time,
            subject=subject,
        )

        return composer_ab_variant

