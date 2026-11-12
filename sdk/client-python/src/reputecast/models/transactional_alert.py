from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_alert_kind import TransactionalAlertKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.alert_domain import AlertDomain
  from ..models.transactional_ref import TransactionalRef





T = TypeVar("T", bound="TransactionalAlert")



@_attrs_define
class TransactionalAlert:
    """ 
        Attributes:
            id (str):
            kind (TransactionalAlertKind):
            started_at (datetime.datetime):
            threshold (float):
            value (float): bounce rate %, failed messages, median seconds or failed-over messages
            bounced (int | Unset):
            domains (list[AlertDomain] | Unset):
            ended_at (datetime.datetime | Unset):
            fallback (str | Unset):
            hard (int | Unset):
            lost (int | Unset):
            median_ms (int | Unset):
            messages (int | Unset):
            provider (TransactionalRef | Unset):
            seen_at (datetime.datetime | Unset):
            sent (int | Unset):
            template (TransactionalRef | Unset):
     """

    id: str
    kind: TransactionalAlertKind
    started_at: datetime.datetime
    threshold: float
    value: float
    bounced: int | Unset = UNSET
    domains: list[AlertDomain] | Unset = UNSET
    ended_at: datetime.datetime | Unset = UNSET
    fallback: str | Unset = UNSET
    hard: int | Unset = UNSET
    lost: int | Unset = UNSET
    median_ms: int | Unset = UNSET
    messages: int | Unset = UNSET
    provider: TransactionalRef | Unset = UNSET
    seen_at: datetime.datetime | Unset = UNSET
    sent: int | Unset = UNSET
    template: TransactionalRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.alert_domain import AlertDomain # noqa: PLC0415
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        id = self.id

        kind = self.kind.value

        started_at = self.started_at.isoformat()

        threshold = self.threshold

        value = self.value

        bounced = self.bounced

        domains: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.domains, Unset):
            domains = []
            for domains_item_data in self.domains:
                domains_item = domains_item_data.to_dict()
                domains.append(domains_item)



        ended_at: str | Unset = UNSET
        if not isinstance(self.ended_at, Unset):
            ended_at = self.ended_at.isoformat()

        fallback = self.fallback

        hard = self.hard

        lost = self.lost

        median_ms = self.median_ms

        messages = self.messages

        provider: dict[str, Any] | Unset = UNSET
        if not isinstance(self.provider, Unset):
            provider = self.provider.to_dict()

        seen_at: str | Unset = UNSET
        if not isinstance(self.seen_at, Unset):
            seen_at = self.seen_at.isoformat()

        sent = self.sent

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
            "startedAt": started_at,
            "threshold": threshold,
            "value": value,
        })
        if bounced is not UNSET:
            field_dict["bounced"] = bounced
        if domains is not UNSET:
            field_dict["domains"] = domains
        if ended_at is not UNSET:
            field_dict["endedAt"] = ended_at
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if hard is not UNSET:
            field_dict["hard"] = hard
        if lost is not UNSET:
            field_dict["lost"] = lost
        if median_ms is not UNSET:
            field_dict["medianMs"] = median_ms
        if messages is not UNSET:
            field_dict["messages"] = messages
        if provider is not UNSET:
            field_dict["provider"] = provider
        if seen_at is not UNSET:
            field_dict["seenAt"] = seen_at
        if sent is not UNSET:
            field_dict["sent"] = sent
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.alert_domain import AlertDomain # noqa: PLC0415
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        kind = TransactionalAlertKind(d.pop("kind"))




        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        threshold = d.pop("threshold")

        value = d.pop("value")

        bounced = d.pop("bounced", UNSET)

        _domains = d.pop("domains", UNSET)
        domains: list[AlertDomain] | Unset = UNSET
        if _domains is not UNSET:
            domains = []
            for domains_item_data in _domains:
                domains_item = AlertDomain.from_dict(domains_item_data)



                domains.append(domains_item)


        _ended_at = d.pop("endedAt", UNSET)
        ended_at: datetime.datetime | Unset
        if isinstance(_ended_at,  Unset):
            ended_at = UNSET
        else:
            ended_at = datetime.datetime.fromisoformat(_ended_at)




        fallback = d.pop("fallback", UNSET)

        hard = d.pop("hard", UNSET)

        lost = d.pop("lost", UNSET)

        median_ms = d.pop("medianMs", UNSET)

        messages = d.pop("messages", UNSET)

        _provider = d.pop("provider", UNSET)
        provider: TransactionalRef | Unset
        if isinstance(_provider,  Unset):
            provider = UNSET
        else:
            provider = TransactionalRef.from_dict(_provider)




        _seen_at = d.pop("seenAt", UNSET)
        seen_at: datetime.datetime | Unset
        if isinstance(_seen_at,  Unset):
            seen_at = UNSET
        else:
            seen_at = datetime.datetime.fromisoformat(_seen_at)




        sent = d.pop("sent", UNSET)

        _template = d.pop("template", UNSET)
        template: TransactionalRef | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = TransactionalRef.from_dict(_template)




        transactional_alert = cls(
            id=id,
            kind=kind,
            started_at=started_at,
            threshold=threshold,
            value=value,
            bounced=bounced,
            domains=domains,
            ended_at=ended_at,
            fallback=fallback,
            hard=hard,
            lost=lost,
            median_ms=median_ms,
            messages=messages,
            provider=provider,
            seen_at=seen_at,
            sent=sent,
            template=template,
        )

        return transactional_alert

