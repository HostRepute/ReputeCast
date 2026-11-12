from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsOutageView")



@_attrs_define
class SmsOutageView:
    """ 
        Attributes:
            countries (list[str]):
            failures (int):
            has_fallback (bool):
            held (int):
            provider (str):
            provider_id (str):
            status (str):
            told (list[str]):
            answer (str | Unset):
            last_accepted (datetime.datetime | Unset):
            next_try (datetime.datetime | Unset):
            since (datetime.datetime | Unset):
     """

    countries: list[str]
    failures: int
    has_fallback: bool
    held: int
    provider: str
    provider_id: str
    status: str
    told: list[str]
    answer: str | Unset = UNSET
    last_accepted: datetime.datetime | Unset = UNSET
    next_try: datetime.datetime | Unset = UNSET
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        countries = self.countries



        failures = self.failures

        has_fallback = self.has_fallback

        held = self.held

        provider = self.provider

        provider_id = self.provider_id

        status = self.status

        told = self.told



        answer = self.answer

        last_accepted: str | Unset = UNSET
        if not isinstance(self.last_accepted, Unset):
            last_accepted = self.last_accepted.isoformat()

        next_try: str | Unset = UNSET
        if not isinstance(self.next_try, Unset):
            next_try = self.next_try.isoformat()

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "failures": failures,
            "hasFallback": has_fallback,
            "held": held,
            "provider": provider,
            "providerId": provider_id,
            "status": status,
            "told": told,
        })
        if answer is not UNSET:
            field_dict["answer"] = answer
        if last_accepted is not UNSET:
            field_dict["lastAccepted"] = last_accepted
        if next_try is not UNSET:
            field_dict["nextTry"] = next_try
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        failures = d.pop("failures")

        has_fallback = d.pop("hasFallback")

        held = d.pop("held")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        status = d.pop("status")

        told = cast(list[str], d.pop("told"))


        answer = d.pop("answer", UNSET)

        _last_accepted = d.pop("lastAccepted", UNSET)
        last_accepted: datetime.datetime | Unset
        if isinstance(_last_accepted,  Unset):
            last_accepted = UNSET
        else:
            last_accepted = datetime.datetime.fromisoformat(_last_accepted)




        _next_try = d.pop("nextTry", UNSET)
        next_try: datetime.datetime | Unset
        if isinstance(_next_try,  Unset):
            next_try = UNSET
        else:
            next_try = datetime.datetime.fromisoformat(_next_try)




        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        sms_outage_view = cls(
            countries=countries,
            failures=failures,
            has_fallback=has_fallback,
            held=held,
            provider=provider,
            provider_id=provider_id,
            status=status,
            told=told,
            answer=answer,
            last_accepted=last_accepted,
            next_try=next_try,
            since=since,
        )

        return sms_outage_view

