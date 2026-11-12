from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_event_kind import SmsEventKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsEvent")



@_attrs_define
class SmsEvent:
    """ 
        Attributes:
            adapter (str):
            at (datetime.datetime):
            filtered (int):
            id (str):
            kind (SmsEventKind):
            lists (list[str]):
            normalized (str):
            number (str):
            provider (str):
            provider_answered (bool):
            raw (str):
            replied (bool):
            signature_ok (bool):
            source (str):
            suppressed (bool):
            temporary (bool):
            code (str | Unset):
            error (str | Unset):
            keyword (str | Unset):
            message_ref (str | Unset):
            reason (str | Unset):
     """

    adapter: str
    at: datetime.datetime
    filtered: int
    id: str
    kind: SmsEventKind
    lists: list[str]
    normalized: str
    number: str
    provider: str
    provider_answered: bool
    raw: str
    replied: bool
    signature_ok: bool
    source: str
    suppressed: bool
    temporary: bool
    code: str | Unset = UNSET
    error: str | Unset = UNSET
    keyword: str | Unset = UNSET
    message_ref: str | Unset = UNSET
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        at = self.at.isoformat()

        filtered = self.filtered

        id = self.id

        kind = self.kind.value

        lists = self.lists



        normalized = self.normalized

        number = self.number

        provider = self.provider

        provider_answered = self.provider_answered

        raw = self.raw

        replied = self.replied

        signature_ok = self.signature_ok

        source = self.source

        suppressed = self.suppressed

        temporary = self.temporary

        code = self.code

        error = self.error

        keyword = self.keyword

        message_ref = self.message_ref

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "at": at,
            "filtered": filtered,
            "id": id,
            "kind": kind,
            "lists": lists,
            "normalized": normalized,
            "number": number,
            "provider": provider,
            "providerAnswered": provider_answered,
            "raw": raw,
            "replied": replied,
            "signatureOk": signature_ok,
            "source": source,
            "suppressed": suppressed,
            "temporary": temporary,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if error is not UNSET:
            field_dict["error"] = error
        if keyword is not UNSET:
            field_dict["keyword"] = keyword
        if message_ref is not UNSET:
            field_dict["messageRef"] = message_ref
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        filtered = d.pop("filtered")

        id = d.pop("id")

        kind = SmsEventKind(d.pop("kind"))




        lists = cast(list[str], d.pop("lists"))


        normalized = d.pop("normalized")

        number = d.pop("number")

        provider = d.pop("provider")

        provider_answered = d.pop("providerAnswered")

        raw = d.pop("raw")

        replied = d.pop("replied")

        signature_ok = d.pop("signatureOk")

        source = d.pop("source")

        suppressed = d.pop("suppressed")

        temporary = d.pop("temporary")

        code = d.pop("code", UNSET)

        error = d.pop("error", UNSET)

        keyword = d.pop("keyword", UNSET)

        message_ref = d.pop("messageRef", UNSET)

        reason = d.pop("reason", UNSET)

        sms_event = cls(
            adapter=adapter,
            at=at,
            filtered=filtered,
            id=id,
            kind=kind,
            lists=lists,
            normalized=normalized,
            number=number,
            provider=provider,
            provider_answered=provider_answered,
            raw=raw,
            replied=replied,
            signature_ok=signature_ok,
            source=source,
            suppressed=suppressed,
            temporary=temporary,
            code=code,
            error=error,
            keyword=keyword,
            message_ref=message_ref,
            reason=reason,
        )

        return sms_event

