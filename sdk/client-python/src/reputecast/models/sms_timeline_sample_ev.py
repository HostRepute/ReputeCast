from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsTimelineSampleEv")



@_attrs_define
class SmsTimelineSampleEv:
    """ 
        Attributes:
            at (datetime.datetime):
            kind (str):
            body (str | Unset):
            link (str | Unset):
            provider (str | Unset):
            seconds (float | Unset):
            sender (str | Unset):
            target (str | Unset):
     """

    at: datetime.datetime
    kind: str
    body: str | Unset = UNSET
    link: str | Unset = UNSET
    provider: str | Unset = UNSET
    seconds: float | Unset = UNSET
    sender: str | Unset = UNSET
    target: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        kind = self.kind

        body = self.body

        link = self.link

        provider = self.provider

        seconds = self.seconds

        sender = self.sender

        target = self.target


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "kind": kind,
        })
        if body is not UNSET:
            field_dict["body"] = body
        if link is not UNSET:
            field_dict["link"] = link
        if provider is not UNSET:
            field_dict["provider"] = provider
        if seconds is not UNSET:
            field_dict["seconds"] = seconds
        if sender is not UNSET:
            field_dict["sender"] = sender
        if target is not UNSET:
            field_dict["target"] = target

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        kind = d.pop("kind")

        body = d.pop("body", UNSET)

        link = d.pop("link", UNSET)

        provider = d.pop("provider", UNSET)

        seconds = d.pop("seconds", UNSET)

        sender = d.pop("sender", UNSET)

        target = d.pop("target", UNSET)

        sms_timeline_sample_ev = cls(
            at=at,
            kind=kind,
            body=body,
            link=link,
            provider=provider,
            seconds=seconds,
            sender=sender,
            target=target,
        )

        return sms_timeline_sample_ev

