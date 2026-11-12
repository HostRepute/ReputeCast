from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_checklist_step import EarnChecklistStep
  from ..models.earn_member_row import EarnMemberRow





T = TypeVar("T", bound="EarnChecklist")



@_attrs_define
class EarnChecklist:
    """ 
        Attributes:
            can_edit (bool):
            can_manage (bool):
            done (int):
            launched (bool):
            steps (list[EarnChecklistStep]):
            test_checkouts (list[EarnMemberRow]):
            upgrade_host (str):
            test_checkout_url (str | Unset):
     """

    can_edit: bool
    can_manage: bool
    done: int
    launched: bool
    steps: list[EarnChecklistStep]
    test_checkouts: list[EarnMemberRow]
    upgrade_host: str
    test_checkout_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_checklist_step import EarnChecklistStep # noqa: PLC0415
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        can_edit = self.can_edit

        can_manage = self.can_manage

        done = self.done

        launched = self.launched

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)



        test_checkouts = []
        for test_checkouts_item_data in self.test_checkouts:
            test_checkouts_item = test_checkouts_item_data.to_dict()
            test_checkouts.append(test_checkouts_item)



        upgrade_host = self.upgrade_host

        test_checkout_url = self.test_checkout_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canManage": can_manage,
            "done": done,
            "launched": launched,
            "steps": steps,
            "testCheckouts": test_checkouts,
            "upgradeHost": upgrade_host,
        })
        if test_checkout_url is not UNSET:
            field_dict["testCheckoutUrl"] = test_checkout_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_checklist_step import EarnChecklistStep # noqa: PLC0415
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_manage = d.pop("canManage")

        done = d.pop("done")

        launched = d.pop("launched")

        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = EarnChecklistStep.from_dict(steps_item_data)



            steps.append(steps_item)


        test_checkouts = []
        _test_checkouts = d.pop("testCheckouts")
        for test_checkouts_item_data in (_test_checkouts):
            test_checkouts_item = EarnMemberRow.from_dict(test_checkouts_item_data)



            test_checkouts.append(test_checkouts_item)


        upgrade_host = d.pop("upgradeHost")

        test_checkout_url = d.pop("testCheckoutUrl", UNSET)

        earn_checklist = cls(
            can_edit=can_edit,
            can_manage=can_manage,
            done=done,
            launched=launched,
            steps=steps,
            test_checkouts=test_checkouts,
            upgrade_host=upgrade_host,
            test_checkout_url=test_checkout_url,
        )

        return earn_checklist

