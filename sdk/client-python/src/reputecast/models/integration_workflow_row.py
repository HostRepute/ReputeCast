from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.workflow_use import WorkflowUse





T = TypeVar("T", bound="IntegrationWorkflowRow")



@_attrs_define
class IntegrationWorkflowRow:
    """ 
        Attributes:
            calls7d (int):
            consent_kept_7_d (int):
            deliveries7d (int):
            errors7d (int):
            key (str):
            last_run_at (datetime.datetime):
            name (str):
            trigger (bool):
            unnamed (bool):
            uses (list[WorkflowUse]):
            fixed_at (datetime.datetime | Unset):
     """

    calls7d: int
    consent_kept_7_d: int
    deliveries7d: int
    errors7d: int
    key: str
    last_run_at: datetime.datetime
    name: str
    trigger: bool
    unnamed: bool
    uses: list[WorkflowUse]
    fixed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.workflow_use import WorkflowUse # noqa: PLC0415
        calls7d = self.calls7d

        consent_kept_7_d = self.consent_kept_7_d

        deliveries7d = self.deliveries7d

        errors7d = self.errors7d

        key = self.key

        last_run_at = self.last_run_at.isoformat()

        name = self.name

        trigger = self.trigger

        unnamed = self.unnamed

        uses = []
        for uses_item_data in self.uses:
            uses_item = uses_item_data.to_dict()
            uses.append(uses_item)



        fixed_at: str | Unset = UNSET
        if not isinstance(self.fixed_at, Unset):
            fixed_at = self.fixed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "calls7d": calls7d,
            "consentKept7d": consent_kept_7_d,
            "deliveries7d": deliveries7d,
            "errors7d": errors7d,
            "key": key,
            "lastRunAt": last_run_at,
            "name": name,
            "trigger": trigger,
            "unnamed": unnamed,
            "uses": uses,
        })
        if fixed_at is not UNSET:
            field_dict["fixedAt"] = fixed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_use import WorkflowUse # noqa: PLC0415
        d = dict(src_dict)
        calls7d = d.pop("calls7d")

        consent_kept_7_d = d.pop("consentKept7d")

        deliveries7d = d.pop("deliveries7d")

        errors7d = d.pop("errors7d")

        key = d.pop("key")

        last_run_at = datetime.datetime.fromisoformat(d.pop("lastRunAt"))




        name = d.pop("name")

        trigger = d.pop("trigger")

        unnamed = d.pop("unnamed")

        uses = []
        _uses = d.pop("uses")
        for uses_item_data in (_uses):
            uses_item = WorkflowUse.from_dict(uses_item_data)



            uses.append(uses_item)


        _fixed_at = d.pop("fixedAt", UNSET)
        fixed_at: datetime.datetime | Unset
        if isinstance(_fixed_at,  Unset):
            fixed_at = UNSET
        else:
            fixed_at = datetime.datetime.fromisoformat(_fixed_at)




        integration_workflow_row = cls(
            calls7d=calls7d,
            consent_kept_7_d=consent_kept_7_d,
            deliveries7d=deliveries7d,
            errors7d=errors7d,
            key=key,
            last_run_at=last_run_at,
            name=name,
            trigger=trigger,
            unnamed=unnamed,
            uses=uses,
            fixed_at=fixed_at,
        )

        return integration_workflow_row

