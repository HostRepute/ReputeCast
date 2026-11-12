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
  from ..models.composer_readiness_group import ComposerReadinessGroup





T = TypeVar("T", bound="ComposerReadiness")



@_attrs_define
class ComposerReadiness:
    """ 
        Attributes:
            checked_at (datetime.datetime | Unset):
            ran_at (datetime.datetime | Unset):
            reputation (ComposerReadinessGroup | Unset):
            rules (ComposerReadinessGroup | Unset):
     """

    checked_at: datetime.datetime | Unset = UNSET
    ran_at: datetime.datetime | Unset = UNSET
    reputation: ComposerReadinessGroup | Unset = UNSET
    rules: ComposerReadinessGroup | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_readiness_group import ComposerReadinessGroup # noqa: PLC0415
        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        ran_at: str | Unset = UNSET
        if not isinstance(self.ran_at, Unset):
            ran_at = self.ran_at.isoformat()

        reputation: dict[str, Any] | Unset = UNSET
        if not isinstance(self.reputation, Unset):
            reputation = self.reputation.to_dict()

        rules: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rules, Unset):
            rules = self.rules.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if ran_at is not UNSET:
            field_dict["ranAt"] = ran_at
        if reputation is not UNSET:
            field_dict["reputation"] = reputation
        if rules is not UNSET:
            field_dict["rules"] = rules

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_readiness_group import ComposerReadinessGroup # noqa: PLC0415
        d = dict(src_dict)
        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _ran_at = d.pop("ranAt", UNSET)
        ran_at: datetime.datetime | Unset
        if isinstance(_ran_at,  Unset):
            ran_at = UNSET
        else:
            ran_at = datetime.datetime.fromisoformat(_ran_at)




        _reputation = d.pop("reputation", UNSET)
        reputation: ComposerReadinessGroup | Unset
        if isinstance(_reputation,  Unset):
            reputation = UNSET
        else:
            reputation = ComposerReadinessGroup.from_dict(_reputation)




        _rules = d.pop("rules", UNSET)
        rules: ComposerReadinessGroup | Unset
        if isinstance(_rules,  Unset):
            rules = UNSET
        else:
            rules = ComposerReadinessGroup.from_dict(_rules)




        composer_readiness = cls(
            checked_at=checked_at,
            ran_at=ran_at,
            reputation=reputation,
            rules=rules,
        )

        return composer_readiness

