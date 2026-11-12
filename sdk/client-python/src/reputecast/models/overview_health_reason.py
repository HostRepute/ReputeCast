from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_health_reason_key import OverviewHealthReasonKey
from ..models.overview_health_reason_tone import OverviewHealthReasonTone
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.overview_health_action import OverviewHealthAction





T = TypeVar("T", bound="OverviewHealthReason")



@_attrs_define
class OverviewHealthReason:
    """ 
        Attributes:
            key (OverviewHealthReasonKey):
            text (str):
            tone (OverviewHealthReasonTone):
            action (OverviewHealthAction | Unset):
     """

    key: OverviewHealthReasonKey
    text: str
    tone: OverviewHealthReasonTone
    action: OverviewHealthAction | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_health_action import OverviewHealthAction # noqa: PLC0415
        key = self.key.value

        text = self.text

        tone = self.tone.value

        action: dict[str, Any] | Unset = UNSET
        if not isinstance(self.action, Unset):
            action = self.action.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "text": text,
            "tone": tone,
        })
        if action is not UNSET:
            field_dict["action"] = action

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_health_action import OverviewHealthAction # noqa: PLC0415
        d = dict(src_dict)
        key = OverviewHealthReasonKey(d.pop("key"))




        text = d.pop("text")

        tone = OverviewHealthReasonTone(d.pop("tone"))




        _action = d.pop("action", UNSET)
        action: OverviewHealthAction | Unset
        if isinstance(_action,  Unset):
            action = UNSET
        else:
            action = OverviewHealthAction.from_dict(_action)




        overview_health_reason = cls(
            key=key,
            text=text,
            tone=tone,
            action=action,
        )

        return overview_health_reason

