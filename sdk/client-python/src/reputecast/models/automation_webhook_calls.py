from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_webhook_call import AutomationWebhookCall





T = TypeVar("T", bound="AutomationWebhookCalls")



@_attrs_define
class AutomationWebhookCalls:
    """ 
        Attributes:
            calls (list[AutomationWebhookCall]):
     """

    calls: list[AutomationWebhookCall]





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_webhook_call import AutomationWebhookCall # noqa: PLC0415
        calls = []
        for calls_item_data in self.calls:
            calls_item = calls_item_data.to_dict()
            calls.append(calls_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "calls": calls,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_webhook_call import AutomationWebhookCall # noqa: PLC0415
        d = dict(src_dict)
        calls = []
        _calls = d.pop("calls")
        for calls_item_data in (_calls):
            calls_item = AutomationWebhookCall.from_dict(calls_item_data)



            calls.append(calls_item)


        automation_webhook_calls = cls(
            calls=calls,
        )

        return automation_webhook_calls

