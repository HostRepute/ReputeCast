from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalFirstRun")



@_attrs_define
class TransactionalFirstRun:
    """ 
        Attributes:
            keys (int):
            messages (int):
            published (int):
            senders (int): Senders on Transactional domains
            templates (int):
            verified_sender (bool):
     """

    keys: int
    messages: int
    published: int
    senders: int
    templates: int
    verified_sender: bool





    def to_dict(self) -> dict[str, Any]:
        keys = self.keys

        messages = self.messages

        published = self.published

        senders = self.senders

        templates = self.templates

        verified_sender = self.verified_sender


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "keys": keys,
            "messages": messages,
            "published": published,
            "senders": senders,
            "templates": templates,
            "verifiedSender": verified_sender,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        keys = d.pop("keys")

        messages = d.pop("messages")

        published = d.pop("published")

        senders = d.pop("senders")

        templates = d.pop("templates")

        verified_sender = d.pop("verifiedSender")

        transactional_first_run = cls(
            keys=keys,
            messages=messages,
            published=published,
            senders=senders,
            templates=templates,
            verified_sender=verified_sender,
        )

        return transactional_first_run

