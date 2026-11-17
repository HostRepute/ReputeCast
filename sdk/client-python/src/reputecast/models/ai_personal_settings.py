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
  from ..models.ai_account_view import AIAccountView
  from ..models.ai_feature_default import AIFeatureDefault
  from ..models.ai_personal_person import AIPersonalPerson
  from ..models.ai_personal_usage_view import AIPersonalUsageView





T = TypeVar("T", bound="AIPersonalSettings")



@_attrs_define
class AIPersonalSettings:
    """ 
        Attributes:
            accounts (list[AIAccountView]):
            can_add (bool):
            cap_usd (float):
            choices (list[AIFeatureDefault]):
            enabled (bool):
            people (list[AIPersonalPerson]):
            usage (AIPersonalUsageView):
            cap_set_at (datetime.datetime | Unset):
            cap_set_by (str | Unset):
     """

    accounts: list[AIAccountView]
    can_add: bool
    cap_usd: float
    choices: list[AIFeatureDefault]
    enabled: bool
    people: list[AIPersonalPerson]
    usage: AIPersonalUsageView
    cap_set_at: datetime.datetime | Unset = UNSET
    cap_set_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_account_view import AIAccountView # noqa: PLC0415
        from ..models.ai_feature_default import AIFeatureDefault # noqa: PLC0415
        from ..models.ai_personal_person import AIPersonalPerson # noqa: PLC0415
        from ..models.ai_personal_usage_view import AIPersonalUsageView # noqa: PLC0415
        accounts = []
        for accounts_item_data in self.accounts:
            accounts_item = accounts_item_data.to_dict()
            accounts.append(accounts_item)



        can_add = self.can_add

        cap_usd = self.cap_usd

        choices = []
        for choices_item_data in self.choices:
            choices_item = choices_item_data.to_dict()
            choices.append(choices_item)



        enabled = self.enabled

        people = []
        for people_item_data in self.people:
            people_item = people_item_data.to_dict()
            people.append(people_item)



        usage = self.usage.to_dict()

        cap_set_at: str | Unset = UNSET
        if not isinstance(self.cap_set_at, Unset):
            cap_set_at = self.cap_set_at.isoformat()

        cap_set_by = self.cap_set_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accounts": accounts,
            "canAdd": can_add,
            "capUsd": cap_usd,
            "choices": choices,
            "enabled": enabled,
            "people": people,
            "usage": usage,
        })
        if cap_set_at is not UNSET:
            field_dict["capSetAt"] = cap_set_at
        if cap_set_by is not UNSET:
            field_dict["capSetBy"] = cap_set_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_account_view import AIAccountView # noqa: PLC0415
        from ..models.ai_feature_default import AIFeatureDefault # noqa: PLC0415
        from ..models.ai_personal_person import AIPersonalPerson # noqa: PLC0415
        from ..models.ai_personal_usage_view import AIPersonalUsageView # noqa: PLC0415
        d = dict(src_dict)
        accounts = []
        _accounts = d.pop("accounts")
        for accounts_item_data in (_accounts):
            accounts_item = AIAccountView.from_dict(accounts_item_data)



            accounts.append(accounts_item)


        can_add = d.pop("canAdd")

        cap_usd = d.pop("capUsd")

        choices = []
        _choices = d.pop("choices")
        for choices_item_data in (_choices):
            choices_item = AIFeatureDefault.from_dict(choices_item_data)



            choices.append(choices_item)


        enabled = d.pop("enabled")

        people = []
        _people = d.pop("people")
        for people_item_data in (_people):
            people_item = AIPersonalPerson.from_dict(people_item_data)



            people.append(people_item)


        usage = AIPersonalUsageView.from_dict(d.pop("usage"))




        _cap_set_at = d.pop("capSetAt", UNSET)
        cap_set_at: datetime.datetime | Unset
        if isinstance(_cap_set_at,  Unset):
            cap_set_at = UNSET
        else:
            cap_set_at = datetime.datetime.fromisoformat(_cap_set_at)




        cap_set_by = d.pop("capSetBy", UNSET)

        ai_personal_settings = cls(
            accounts=accounts,
            can_add=can_add,
            cap_usd=cap_usd,
            choices=choices,
            enabled=enabled,
            people=people,
            usage=usage,
            cap_set_at=cap_set_at,
            cap_set_by=cap_set_by,
        )

        return ai_personal_settings

