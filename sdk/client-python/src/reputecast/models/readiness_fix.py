from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_fix_action import ReadinessFixAction
from ..models.readiness_fix_kind import ReadinessFixKind
from ..models.readiness_fix_tone import ReadinessFixTone
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_fix_params import ReadinessFixParams
  from ..models.readiness_tech_row import ReadinessTechRow





T = TypeVar("T", bound="ReadinessFix")



@_attrs_define
class ReadinessFix:
    """ 
        Attributes:
            kind (ReadinessFixKind):
            label (str):
            action (ReadinessFixAction | Unset):
            effects (list[ReadinessTechRow] | Unset):
            href (str | Unset):
            note (str | Unset):
            params (ReadinessFixParams | Unset):
            tone (ReadinessFixTone | Unset):
     """

    kind: ReadinessFixKind
    label: str
    action: ReadinessFixAction | Unset = UNSET
    effects: list[ReadinessTechRow] | Unset = UNSET
    href: str | Unset = UNSET
    note: str | Unset = UNSET
    params: ReadinessFixParams | Unset = UNSET
    tone: ReadinessFixTone | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_fix_params import ReadinessFixParams # noqa: PLC0415
        from ..models.readiness_tech_row import ReadinessTechRow # noqa: PLC0415
        kind = self.kind.value

        label = self.label

        action: str | Unset = UNSET
        if not isinstance(self.action, Unset):
            action = self.action.value


        effects: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.effects, Unset):
            effects = []
            for effects_item_data in self.effects:
                effects_item = effects_item_data.to_dict()
                effects.append(effects_item)



        href = self.href

        note = self.note

        params: dict[str, Any] | Unset = UNSET
        if not isinstance(self.params, Unset):
            params = self.params.to_dict()

        tone: str | Unset = UNSET
        if not isinstance(self.tone, Unset):
            tone = self.tone.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if action is not UNSET:
            field_dict["action"] = action
        if effects is not UNSET:
            field_dict["effects"] = effects
        if href is not UNSET:
            field_dict["href"] = href
        if note is not UNSET:
            field_dict["note"] = note
        if params is not UNSET:
            field_dict["params"] = params
        if tone is not UNSET:
            field_dict["tone"] = tone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_fix_params import ReadinessFixParams # noqa: PLC0415
        from ..models.readiness_tech_row import ReadinessTechRow # noqa: PLC0415
        d = dict(src_dict)
        kind = ReadinessFixKind(d.pop("kind"))




        label = d.pop("label")

        _action = d.pop("action", UNSET)
        action: ReadinessFixAction | Unset
        if isinstance(_action,  Unset):
            action = UNSET
        else:
            action = ReadinessFixAction(_action)




        _effects = d.pop("effects", UNSET)
        effects: list[ReadinessTechRow] | Unset = UNSET
        if _effects is not UNSET:
            effects = []
            for effects_item_data in _effects:
                effects_item = ReadinessTechRow.from_dict(effects_item_data)



                effects.append(effects_item)


        href = d.pop("href", UNSET)

        note = d.pop("note", UNSET)

        _params = d.pop("params", UNSET)
        params: ReadinessFixParams | Unset
        if isinstance(_params,  Unset):
            params = UNSET
        else:
            params = ReadinessFixParams.from_dict(_params)




        _tone = d.pop("tone", UNSET)
        tone: ReadinessFixTone | Unset
        if isinstance(_tone,  Unset):
            tone = UNSET
        else:
            tone = ReadinessFixTone(_tone)




        readiness_fix = cls(
            kind=kind,
            label=label,
            action=action,
            effects=effects,
            href=href,
            note=note,
            params=params,
            tone=tone,
        )

        return readiness_fix

