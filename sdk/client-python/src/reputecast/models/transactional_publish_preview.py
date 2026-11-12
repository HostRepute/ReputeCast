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
  from ..models.transactional_breaking_change import TransactionalBreakingChange
  from ..models.transactional_category_ref import TransactionalCategoryRef
  from ..models.transactional_checks import TransactionalChecks
  from ..models.transactional_diff import TransactionalDiff
  from ..models.transactional_publish_approval import TransactionalPublishApproval
  from ..models.transactional_publish_language import TransactionalPublishLanguage
  from ..models.transactional_test_render import TransactionalTestRender





T = TypeVar("T", bound="TransactionalPublishPreview")



@_attrs_define
class TransactionalPublishPreview:
    """ 
        Attributes:
            approval (TransactionalPublishApproval):
            blocking (list[str]):
            breaking (TransactionalBreakingChange):
            can_publish (bool):
            checks (TransactionalChecks):
            diff (TransactionalDiff):
            draft_revision (int):
            draft_saved_at (datetime.datetime):
            draft_saved_by (str):
            key (str):
            languages (list[TransactionalPublishLanguage]):
            name (str):
            next_version (int):
            template_id (str):
            test_renders (list[TransactionalTestRender]):
            category (TransactionalCategoryRef | Unset):
            live_by (str | Unset):
            live_since (datetime.datetime | Unset):
            live_version (int | Unset):
     """

    approval: TransactionalPublishApproval
    blocking: list[str]
    breaking: TransactionalBreakingChange
    can_publish: bool
    checks: TransactionalChecks
    diff: TransactionalDiff
    draft_revision: int
    draft_saved_at: datetime.datetime
    draft_saved_by: str
    key: str
    languages: list[TransactionalPublishLanguage]
    name: str
    next_version: int
    template_id: str
    test_renders: list[TransactionalTestRender]
    category: TransactionalCategoryRef | Unset = UNSET
    live_by: str | Unset = UNSET
    live_since: datetime.datetime | Unset = UNSET
    live_version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_breaking_change import TransactionalBreakingChange # noqa: PLC0415
        from ..models.transactional_category_ref import TransactionalCategoryRef # noqa: PLC0415
        from ..models.transactional_checks import TransactionalChecks # noqa: PLC0415
        from ..models.transactional_diff import TransactionalDiff # noqa: PLC0415
        from ..models.transactional_publish_approval import TransactionalPublishApproval # noqa: PLC0415
        from ..models.transactional_publish_language import TransactionalPublishLanguage # noqa: PLC0415
        from ..models.transactional_test_render import TransactionalTestRender # noqa: PLC0415
        approval = self.approval.to_dict()

        blocking = self.blocking



        breaking = self.breaking.to_dict()

        can_publish = self.can_publish

        checks = self.checks.to_dict()

        diff = self.diff.to_dict()

        draft_revision = self.draft_revision

        draft_saved_at = self.draft_saved_at.isoformat()

        draft_saved_by = self.draft_saved_by

        key = self.key

        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        name = self.name

        next_version = self.next_version

        template_id = self.template_id

        test_renders = []
        for test_renders_item_data in self.test_renders:
            test_renders_item = test_renders_item_data.to_dict()
            test_renders.append(test_renders_item)



        category: dict[str, Any] | Unset = UNSET
        if not isinstance(self.category, Unset):
            category = self.category.to_dict()

        live_by = self.live_by

        live_since: str | Unset = UNSET
        if not isinstance(self.live_since, Unset):
            live_since = self.live_since.isoformat()

        live_version = self.live_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approval": approval,
            "blocking": blocking,
            "breaking": breaking,
            "canPublish": can_publish,
            "checks": checks,
            "diff": diff,
            "draftRevision": draft_revision,
            "draftSavedAt": draft_saved_at,
            "draftSavedBy": draft_saved_by,
            "key": key,
            "languages": languages,
            "name": name,
            "nextVersion": next_version,
            "templateId": template_id,
            "testRenders": test_renders,
        })
        if category is not UNSET:
            field_dict["category"] = category
        if live_by is not UNSET:
            field_dict["liveBy"] = live_by
        if live_since is not UNSET:
            field_dict["liveSince"] = live_since
        if live_version is not UNSET:
            field_dict["liveVersion"] = live_version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_breaking_change import TransactionalBreakingChange # noqa: PLC0415
        from ..models.transactional_category_ref import TransactionalCategoryRef # noqa: PLC0415
        from ..models.transactional_checks import TransactionalChecks # noqa: PLC0415
        from ..models.transactional_diff import TransactionalDiff # noqa: PLC0415
        from ..models.transactional_publish_approval import TransactionalPublishApproval # noqa: PLC0415
        from ..models.transactional_publish_language import TransactionalPublishLanguage # noqa: PLC0415
        from ..models.transactional_test_render import TransactionalTestRender # noqa: PLC0415
        d = dict(src_dict)
        approval = TransactionalPublishApproval.from_dict(d.pop("approval"))




        blocking = cast(list[str], d.pop("blocking"))


        breaking = TransactionalBreakingChange.from_dict(d.pop("breaking"))




        can_publish = d.pop("canPublish")

        checks = TransactionalChecks.from_dict(d.pop("checks"))




        diff = TransactionalDiff.from_dict(d.pop("diff"))




        draft_revision = d.pop("draftRevision")

        draft_saved_at = datetime.datetime.fromisoformat(d.pop("draftSavedAt"))




        draft_saved_by = d.pop("draftSavedBy")

        key = d.pop("key")

        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = TransactionalPublishLanguage.from_dict(languages_item_data)



            languages.append(languages_item)


        name = d.pop("name")

        next_version = d.pop("nextVersion")

        template_id = d.pop("templateId")

        test_renders = []
        _test_renders = d.pop("testRenders")
        for test_renders_item_data in (_test_renders):
            test_renders_item = TransactionalTestRender.from_dict(test_renders_item_data)



            test_renders.append(test_renders_item)


        _category = d.pop("category", UNSET)
        category: TransactionalCategoryRef | Unset
        if isinstance(_category,  Unset):
            category = UNSET
        else:
            category = TransactionalCategoryRef.from_dict(_category)




        live_by = d.pop("liveBy", UNSET)

        _live_since = d.pop("liveSince", UNSET)
        live_since: datetime.datetime | Unset
        if isinstance(_live_since,  Unset):
            live_since = UNSET
        else:
            live_since = datetime.datetime.fromisoformat(_live_since)




        live_version = d.pop("liveVersion", UNSET)

        transactional_publish_preview = cls(
            approval=approval,
            blocking=blocking,
            breaking=breaking,
            can_publish=can_publish,
            checks=checks,
            diff=diff,
            draft_revision=draft_revision,
            draft_saved_at=draft_saved_at,
            draft_saved_by=draft_saved_by,
            key=key,
            languages=languages,
            name=name,
            next_version=next_version,
            template_id=template_id,
            test_renders=test_renders,
            category=category,
            live_by=live_by,
            live_since=live_since,
            live_version=live_version,
        )

        return transactional_publish_preview

