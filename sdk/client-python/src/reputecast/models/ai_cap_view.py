from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_cap_view_kind import AICapViewKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AICapView")



@_attrs_define
class AICapView:
    """ 
        Attributes:
            kind (AICapViewKind):
            tokens (int):
            usd (float):
            set_at (datetime.datetime | Unset):
            set_by (str | Unset):
     """

    kind: AICapViewKind
    tokens: int
    usd: float
    set_at: datetime.datetime | Unset = UNSET
    set_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        tokens = self.tokens

        usd = self.usd

        set_at: str | Unset = UNSET
        if not isinstance(self.set_at, Unset):
            set_at = self.set_at.isoformat()

        set_by = self.set_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "tokens": tokens,
            "usd": usd,
        })
        if set_at is not UNSET:
            field_dict["setAt"] = set_at
        if set_by is not UNSET:
            field_dict["setBy"] = set_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = AICapViewKind(d.pop("kind"))




        tokens = d.pop("tokens")

        usd = d.pop("usd")

        _set_at = d.pop("setAt", UNSET)
        set_at: datetime.datetime | Unset
        if isinstance(_set_at,  Unset):
            set_at = UNSET
        else:
            set_at = datetime.datetime.fromisoformat(_set_at)




        set_by = d.pop("setBy", UNSET)

        ai_cap_view = cls(
            kind=kind,
            tokens=tokens,
            usd=usd,
            set_at=set_at,
            set_by=set_by,
        )

        return ai_cap_view

