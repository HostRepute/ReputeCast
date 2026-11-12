from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sending_rule_applies_to import SendingRuleAppliesTo
from ..models.sending_rule_kind import SendingRuleKind
from ..models.sending_rule_operator import SendingRuleOperator
from ..models.sending_rule_severity import SendingRuleSeverity
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sending_rule_trigger import SendingRuleTrigger





T = TypeVar("T", bound="SendingRule")



@_attrs_define
class SendingRule:
    """ 
        Attributes:
            added_at (datetime.datetime):
            added_by (str):
            applies_to (SendingRuleAppliesTo):
            description (str):
            enabled (bool):
            id (str):
            kind (SendingRuleKind):
            message (str):
            name (str):
            operator (SendingRuleOperator):
            severity (SendingRuleSeverity):
            value (str):
            triggered (SendingRuleTrigger | Unset):
     """

    added_at: datetime.datetime
    added_by: str
    applies_to: SendingRuleAppliesTo
    description: str
    enabled: bool
    id: str
    kind: SendingRuleKind
    message: str
    name: str
    operator: SendingRuleOperator
    severity: SendingRuleSeverity
    value: str
    triggered: SendingRuleTrigger | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sending_rule_trigger import SendingRuleTrigger # noqa: PLC0415
        added_at = self.added_at.isoformat()

        added_by = self.added_by

        applies_to = self.applies_to.value

        description = self.description

        enabled = self.enabled

        id = self.id

        kind = self.kind.value

        message = self.message

        name = self.name

        operator = self.operator.value

        severity = self.severity.value

        value = self.value

        triggered: dict[str, Any] | Unset = UNSET
        if not isinstance(self.triggered, Unset):
            triggered = self.triggered.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addedAt": added_at,
            "addedBy": added_by,
            "appliesTo": applies_to,
            "description": description,
            "enabled": enabled,
            "id": id,
            "kind": kind,
            "message": message,
            "name": name,
            "operator": operator,
            "severity": severity,
            "value": value,
        })
        if triggered is not UNSET:
            field_dict["triggered"] = triggered

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sending_rule_trigger import SendingRuleTrigger # noqa: PLC0415
        d = dict(src_dict)
        added_at = datetime.datetime.fromisoformat(d.pop("addedAt"))




        added_by = d.pop("addedBy")

        applies_to = SendingRuleAppliesTo(d.pop("appliesTo"))




        description = d.pop("description")

        enabled = d.pop("enabled")

        id = d.pop("id")

        kind = SendingRuleKind(d.pop("kind"))




        message = d.pop("message")

        name = d.pop("name")

        operator = SendingRuleOperator(d.pop("operator"))




        severity = SendingRuleSeverity(d.pop("severity"))




        value = d.pop("value")

        _triggered = d.pop("triggered", UNSET)
        triggered: SendingRuleTrigger | Unset
        if isinstance(_triggered,  Unset):
            triggered = UNSET
        else:
            triggered = SendingRuleTrigger.from_dict(_triggered)




        sending_rule = cls(
            added_at=added_at,
            added_by=added_by,
            applies_to=applies_to,
            description=description,
            enabled=enabled,
            id=id,
            kind=kind,
            message=message,
            name=name,
            operator=operator,
            severity=severity,
            value=value,
            triggered=triggered,
        )

        return sending_rule

