from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_pause_view_kind import SmsPauseViewKind
from ..models.sms_pause_view_metric import SmsPauseViewMetric
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsPauseView")



@_attrs_define
class SmsPauseView:
    """ 
        Attributes:
            base (int):
            code_count (int):
            events (int):
            held (int):
            kind (SmsPauseViewKind):
            limit (float):
            value (float):
            at (datetime.datetime | Unset):
            by (str | Unset):
            code (str | Unset):
            metric (SmsPauseViewMetric | Unset):
            provider (str | Unset):
     """

    base: int
    code_count: int
    events: int
    held: int
    kind: SmsPauseViewKind
    limit: float
    value: float
    at: datetime.datetime | Unset = UNSET
    by: str | Unset = UNSET
    code: str | Unset = UNSET
    metric: SmsPauseViewMetric | Unset = UNSET
    provider: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        base = self.base

        code_count = self.code_count

        events = self.events

        held = self.held

        kind = self.kind.value

        limit = self.limit

        value = self.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        by = self.by

        code = self.code

        metric: str | Unset = UNSET
        if not isinstance(self.metric, Unset):
            metric = self.metric.value


        provider = self.provider


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "base": base,
            "codeCount": code_count,
            "events": events,
            "held": held,
            "kind": kind,
            "limit": limit,
            "value": value,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if by is not UNSET:
            field_dict["by"] = by
        if code is not UNSET:
            field_dict["code"] = code
        if metric is not UNSET:
            field_dict["metric"] = metric
        if provider is not UNSET:
            field_dict["provider"] = provider

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base = d.pop("base")

        code_count = d.pop("codeCount")

        events = d.pop("events")

        held = d.pop("held")

        kind = SmsPauseViewKind(d.pop("kind"))




        limit = d.pop("limit")

        value = d.pop("value")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        by = d.pop("by", UNSET)

        code = d.pop("code", UNSET)

        _metric = d.pop("metric", UNSET)
        metric: SmsPauseViewMetric | Unset
        if isinstance(_metric,  Unset):
            metric = UNSET
        else:
            metric = SmsPauseViewMetric(_metric)




        provider = d.pop("provider", UNSET)

        sms_pause_view = cls(
            base=base,
            code_count=code_count,
            events=events,
            held=held,
            kind=kind,
            limit=limit,
            value=value,
            at=at,
            by=by,
            code=code,
            metric=metric,
            provider=provider,
        )

        return sms_pause_view

