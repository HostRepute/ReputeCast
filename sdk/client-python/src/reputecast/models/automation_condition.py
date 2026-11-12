from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_condition_kind import AutomationConditionKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_duration import AutomationDuration





T = TypeVar("T", bound="AutomationCondition")



@_attrs_define
class AutomationCondition:
    """ 
        Attributes:
            kind (AutomationConditionKind):
            event (str | Unset):
            field (str | Unset):
            human_only (bool | Unset):
            list_id (str | Unset):
            op (str | Unset): equals, not_equals, contains, is_set, is_not_set, is_one_of (comma-separated values)
            product_id (str | Unset):
            step (str | Unset):
            store (str | Unset):
            tag (str | Unset):
            url_contains (str | Unset):
            value (str | Unset):
            within (AutomationDuration | Unset):
     """

    kind: AutomationConditionKind
    event: str | Unset = UNSET
    field: str | Unset = UNSET
    human_only: bool | Unset = UNSET
    list_id: str | Unset = UNSET
    op: str | Unset = UNSET
    product_id: str | Unset = UNSET
    step: str | Unset = UNSET
    store: str | Unset = UNSET
    tag: str | Unset = UNSET
    url_contains: str | Unset = UNSET
    value: str | Unset = UNSET
    within: AutomationDuration | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_duration import AutomationDuration # noqa: PLC0415
        kind = self.kind.value

        event = self.event

        field = self.field

        human_only = self.human_only

        list_id = self.list_id

        op = self.op

        product_id = self.product_id

        step = self.step

        store = self.store

        tag = self.tag

        url_contains = self.url_contains

        value = self.value

        within: dict[str, Any] | Unset = UNSET
        if not isinstance(self.within, Unset):
            within = self.within.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if event is not UNSET:
            field_dict["event"] = event
        if field is not UNSET:
            field_dict["field"] = field
        if human_only is not UNSET:
            field_dict["humanOnly"] = human_only
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if op is not UNSET:
            field_dict["op"] = op
        if product_id is not UNSET:
            field_dict["productId"] = product_id
        if step is not UNSET:
            field_dict["step"] = step
        if store is not UNSET:
            field_dict["store"] = store
        if tag is not UNSET:
            field_dict["tag"] = tag
        if url_contains is not UNSET:
            field_dict["urlContains"] = url_contains
        if value is not UNSET:
            field_dict["value"] = value
        if within is not UNSET:
            field_dict["within"] = within

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_duration import AutomationDuration # noqa: PLC0415
        d = dict(src_dict)
        kind = AutomationConditionKind(d.pop("kind"))




        event = d.pop("event", UNSET)

        field = d.pop("field", UNSET)

        human_only = d.pop("humanOnly", UNSET)

        list_id = d.pop("listId", UNSET)

        op = d.pop("op", UNSET)

        product_id = d.pop("productId", UNSET)

        step = d.pop("step", UNSET)

        store = d.pop("store", UNSET)

        tag = d.pop("tag", UNSET)

        url_contains = d.pop("urlContains", UNSET)

        value = d.pop("value", UNSET)

        _within = d.pop("within", UNSET)
        within: AutomationDuration | Unset
        if isinstance(_within,  Unset):
            within = UNSET
        else:
            within = AutomationDuration.from_dict(_within)




        automation_condition = cls(
            kind=kind,
            event=event,
            field=field,
            human_only=human_only,
            list_id=list_id,
            op=op,
            product_id=product_id,
            step=step,
            store=store,
            tag=tag,
            url_contains=url_contains,
            value=value,
            within=within,
        )

        return automation_condition

