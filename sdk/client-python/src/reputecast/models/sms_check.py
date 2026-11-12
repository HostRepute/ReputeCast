from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_check_choice import SmsCheckChoice
from ..models.sms_check_key import SmsCheckKey
from ..models.sms_check_status import SmsCheckStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_check_reasons import SmsCheckReasons
  from ..models.sms_check_sender import SmsCheckSender





T = TypeVar("T", bound="SmsCheck")



@_attrs_define
class SmsCheck:
    """ 
        Attributes:
            after (float):
            alert_pct (int):
            budget (float):
            by_area (int):
            by_country (int):
            characters (int):
            cost (float):
            count (int):
            countries (list[str]):
            key (SmsCheckKey):
            links (int):
            lists (list[str]):
            reasons (SmsCheckReasons):
            remaining (float):
            segments (int):
            senders (list[SmsCheckSender]):
            shorteners (list[str]):
            spent (float):
            status (SmsCheckStatus):
            total (int):
            choice (SmsCheckChoice | Unset):
            cleared_at (datetime.datetime | Unset):
            cleared_by (str | Unset):
            domain (str | Unset):
            encoding (str | Unset):
            local (str | Unset):
            provider (str | Unset):
            registration (str | Unset):
            sender (str | Unset):
            since (datetime.datetime | Unset):
            text (str | Unset):
            verified_on (datetime.datetime | Unset):
            window (str | Unset):
     """

    after: float
    alert_pct: int
    budget: float
    by_area: int
    by_country: int
    characters: int
    cost: float
    count: int
    countries: list[str]
    key: SmsCheckKey
    links: int
    lists: list[str]
    reasons: SmsCheckReasons
    remaining: float
    segments: int
    senders: list[SmsCheckSender]
    shorteners: list[str]
    spent: float
    status: SmsCheckStatus
    total: int
    choice: SmsCheckChoice | Unset = UNSET
    cleared_at: datetime.datetime | Unset = UNSET
    cleared_by: str | Unset = UNSET
    domain: str | Unset = UNSET
    encoding: str | Unset = UNSET
    local: str | Unset = UNSET
    provider: str | Unset = UNSET
    registration: str | Unset = UNSET
    sender: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET
    text: str | Unset = UNSET
    verified_on: datetime.datetime | Unset = UNSET
    window: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_check_reasons import SmsCheckReasons # noqa: PLC0415
        from ..models.sms_check_sender import SmsCheckSender # noqa: PLC0415
        after = self.after

        alert_pct = self.alert_pct

        budget = self.budget

        by_area = self.by_area

        by_country = self.by_country

        characters = self.characters

        cost = self.cost

        count = self.count

        countries = self.countries



        key = self.key.value

        links = self.links

        lists = self.lists



        reasons = self.reasons.to_dict()

        remaining = self.remaining

        segments = self.segments

        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        shorteners = self.shorteners



        spent = self.spent

        status = self.status.value

        total = self.total

        choice: str | Unset = UNSET
        if not isinstance(self.choice, Unset):
            choice = self.choice.value


        cleared_at: str | Unset = UNSET
        if not isinstance(self.cleared_at, Unset):
            cleared_at = self.cleared_at.isoformat()

        cleared_by = self.cleared_by

        domain = self.domain

        encoding = self.encoding

        local = self.local

        provider = self.provider

        registration = self.registration

        sender = self.sender

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()

        text = self.text

        verified_on: str | Unset = UNSET
        if not isinstance(self.verified_on, Unset):
            verified_on = self.verified_on.isoformat()

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after": after,
            "alertPct": alert_pct,
            "budget": budget,
            "byArea": by_area,
            "byCountry": by_country,
            "characters": characters,
            "cost": cost,
            "count": count,
            "countries": countries,
            "key": key,
            "links": links,
            "lists": lists,
            "reasons": reasons,
            "remaining": remaining,
            "segments": segments,
            "senders": senders,
            "shorteners": shorteners,
            "spent": spent,
            "status": status,
            "total": total,
        })
        if choice is not UNSET:
            field_dict["choice"] = choice
        if cleared_at is not UNSET:
            field_dict["clearedAt"] = cleared_at
        if cleared_by is not UNSET:
            field_dict["clearedBy"] = cleared_by
        if domain is not UNSET:
            field_dict["domain"] = domain
        if encoding is not UNSET:
            field_dict["encoding"] = encoding
        if local is not UNSET:
            field_dict["local"] = local
        if provider is not UNSET:
            field_dict["provider"] = provider
        if registration is not UNSET:
            field_dict["registration"] = registration
        if sender is not UNSET:
            field_dict["sender"] = sender
        if since is not UNSET:
            field_dict["since"] = since
        if text is not UNSET:
            field_dict["text"] = text
        if verified_on is not UNSET:
            field_dict["verifiedOn"] = verified_on
        if window is not UNSET:
            field_dict["window"] = window

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_check_reasons import SmsCheckReasons # noqa: PLC0415
        from ..models.sms_check_sender import SmsCheckSender # noqa: PLC0415
        d = dict(src_dict)
        after = d.pop("after")

        alert_pct = d.pop("alertPct")

        budget = d.pop("budget")

        by_area = d.pop("byArea")

        by_country = d.pop("byCountry")

        characters = d.pop("characters")

        cost = d.pop("cost")

        count = d.pop("count")

        countries = cast(list[str], d.pop("countries"))


        key = SmsCheckKey(d.pop("key"))




        links = d.pop("links")

        lists = cast(list[str], d.pop("lists"))


        reasons = SmsCheckReasons.from_dict(d.pop("reasons"))




        remaining = d.pop("remaining")

        segments = d.pop("segments")

        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = SmsCheckSender.from_dict(senders_item_data)



            senders.append(senders_item)


        shorteners = cast(list[str], d.pop("shorteners"))


        spent = d.pop("spent")

        status = SmsCheckStatus(d.pop("status"))




        total = d.pop("total")

        _choice = d.pop("choice", UNSET)
        choice: SmsCheckChoice | Unset
        if isinstance(_choice,  Unset):
            choice = UNSET
        else:
            choice = SmsCheckChoice(_choice)




        _cleared_at = d.pop("clearedAt", UNSET)
        cleared_at: datetime.datetime | Unset
        if isinstance(_cleared_at,  Unset):
            cleared_at = UNSET
        else:
            cleared_at = datetime.datetime.fromisoformat(_cleared_at)




        cleared_by = d.pop("clearedBy", UNSET)

        domain = d.pop("domain", UNSET)

        encoding = d.pop("encoding", UNSET)

        local = d.pop("local", UNSET)

        provider = d.pop("provider", UNSET)

        registration = d.pop("registration", UNSET)

        sender = d.pop("sender", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        text = d.pop("text", UNSET)

        _verified_on = d.pop("verifiedOn", UNSET)
        verified_on: datetime.datetime | Unset
        if isinstance(_verified_on,  Unset):
            verified_on = UNSET
        else:
            verified_on = datetime.datetime.fromisoformat(_verified_on)




        window = d.pop("window", UNSET)

        sms_check = cls(
            after=after,
            alert_pct=alert_pct,
            budget=budget,
            by_area=by_area,
            by_country=by_country,
            characters=characters,
            cost=cost,
            count=count,
            countries=countries,
            key=key,
            links=links,
            lists=lists,
            reasons=reasons,
            remaining=remaining,
            segments=segments,
            senders=senders,
            shorteners=shorteners,
            spent=spent,
            status=status,
            total=total,
            choice=choice,
            cleared_at=cleared_at,
            cleared_by=cleared_by,
            domain=domain,
            encoding=encoding,
            local=local,
            provider=provider,
            registration=registration,
            sender=sender,
            since=since,
            text=text,
            verified_on=verified_on,
            window=window,
        )

        return sms_check

