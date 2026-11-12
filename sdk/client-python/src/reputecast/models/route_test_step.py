from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_test_step_kind import RouteTestStepKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.route_test_split import RouteTestSplit





T = TypeVar("T", bound="RouteTestStep")



@_attrs_define
class RouteTestStep:
    """ 
        Attributes:
            conditions (int):
            kind (RouteTestStepKind):
            suppressed (bool):
            bucket (int | Unset):
            email (str | Unset):
            field (str | Unset):
            provider (str | Unset):
            reason (str | Unset):
            splits (list[RouteTestSplit] | Unset):
            stream (str | Unset):
            value (str | Unset):
            warming (str | Unset):
     """

    conditions: int
    kind: RouteTestStepKind
    suppressed: bool
    bucket: int | Unset = UNSET
    email: str | Unset = UNSET
    field: str | Unset = UNSET
    provider: str | Unset = UNSET
    reason: str | Unset = UNSET
    splits: list[RouteTestSplit] | Unset = UNSET
    stream: str | Unset = UNSET
    value: str | Unset = UNSET
    warming: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.route_test_split import RouteTestSplit # noqa: PLC0415
        conditions = self.conditions

        kind = self.kind.value

        suppressed = self.suppressed

        bucket = self.bucket

        email = self.email

        field = self.field

        provider = self.provider

        reason = self.reason

        splits: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.splits, Unset):
            splits = []
            for splits_item_data in self.splits:
                splits_item = splits_item_data.to_dict()
                splits.append(splits_item)



        stream = self.stream

        value = self.value

        warming = self.warming


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditions": conditions,
            "kind": kind,
            "suppressed": suppressed,
        })
        if bucket is not UNSET:
            field_dict["bucket"] = bucket
        if email is not UNSET:
            field_dict["email"] = email
        if field is not UNSET:
            field_dict["field"] = field
        if provider is not UNSET:
            field_dict["provider"] = provider
        if reason is not UNSET:
            field_dict["reason"] = reason
        if splits is not UNSET:
            field_dict["splits"] = splits
        if stream is not UNSET:
            field_dict["stream"] = stream
        if value is not UNSET:
            field_dict["value"] = value
        if warming is not UNSET:
            field_dict["warming"] = warming

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.route_test_split import RouteTestSplit # noqa: PLC0415
        d = dict(src_dict)
        conditions = d.pop("conditions")

        kind = RouteTestStepKind(d.pop("kind"))




        suppressed = d.pop("suppressed")

        bucket = d.pop("bucket", UNSET)

        email = d.pop("email", UNSET)

        field = d.pop("field", UNSET)

        provider = d.pop("provider", UNSET)

        reason = d.pop("reason", UNSET)

        _splits = d.pop("splits", UNSET)
        splits: list[RouteTestSplit] | Unset = UNSET
        if _splits is not UNSET:
            splits = []
            for splits_item_data in _splits:
                splits_item = RouteTestSplit.from_dict(splits_item_data)



                splits.append(splits_item)


        stream = d.pop("stream", UNSET)

        value = d.pop("value", UNSET)

        warming = d.pop("warming", UNSET)

        route_test_step = cls(
            conditions=conditions,
            kind=kind,
            suppressed=suppressed,
            bucket=bucket,
            email=email,
            field=field,
            provider=provider,
            reason=reason,
            splits=splits,
            stream=stream,
            value=value,
            warming=warming,
        )

        return route_test_step

