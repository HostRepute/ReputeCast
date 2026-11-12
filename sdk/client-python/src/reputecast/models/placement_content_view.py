from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_content_view_kind import PlacementContentViewKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.placement_conditional_version import PlacementConditionalVersion
  from ..models.placement_language import PlacementLanguage
  from ..models.placement_test_value import PlacementTestValue
  from ..models.placement_variant import PlacementVariant





T = TypeVar("T", bound="PlacementContentView")



@_attrs_define
class PlacementContentView:
    """ 
        Attributes:
            conditional (list[PlacementConditionalVersion]):
            id (str):
            kind (PlacementContentViewKind):
            languages (list[PlacementLanguage]):
            name (str):
            sensitive (list[str]):
            test_data (list[str]):
            test_values (list[PlacementTestValue]):
            variants (list[PlacementVariant]):
            audience (str | Unset):
            category (str | Unset):
            conditional_version (int | Unset):
            fallback_greeting (str | Unset):
            key (str | Unset):
            recipients (int | Unset):
            scheduled_at (datetime.datetime | Unset):
            sent_at (datetime.datetime | Unset):
            status (str | Unset):
            version (int | Unset):
     """

    conditional: list[PlacementConditionalVersion]
    id: str
    kind: PlacementContentViewKind
    languages: list[PlacementLanguage]
    name: str
    sensitive: list[str]
    test_data: list[str]
    test_values: list[PlacementTestValue]
    variants: list[PlacementVariant]
    audience: str | Unset = UNSET
    category: str | Unset = UNSET
    conditional_version: int | Unset = UNSET
    fallback_greeting: str | Unset = UNSET
    key: str | Unset = UNSET
    recipients: int | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET
    status: str | Unset = UNSET
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_conditional_version import PlacementConditionalVersion # noqa: PLC0415
        from ..models.placement_language import PlacementLanguage # noqa: PLC0415
        from ..models.placement_test_value import PlacementTestValue # noqa: PLC0415
        from ..models.placement_variant import PlacementVariant # noqa: PLC0415
        conditional = []
        for conditional_item_data in self.conditional:
            conditional_item = conditional_item_data.to_dict()
            conditional.append(conditional_item)



        id = self.id

        kind = self.kind.value

        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        name = self.name

        sensitive = self.sensitive



        test_data = self.test_data



        test_values = []
        for test_values_item_data in self.test_values:
            test_values_item = test_values_item_data.to_dict()
            test_values.append(test_values_item)



        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)



        audience = self.audience

        category = self.category

        conditional_version = self.conditional_version

        fallback_greeting = self.fallback_greeting

        key = self.key

        recipients = self.recipients

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        status = self.status

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "conditional": conditional,
            "id": id,
            "kind": kind,
            "languages": languages,
            "name": name,
            "sensitive": sensitive,
            "testData": test_data,
            "testValues": test_values,
            "variants": variants,
        })
        if audience is not UNSET:
            field_dict["audience"] = audience
        if category is not UNSET:
            field_dict["category"] = category
        if conditional_version is not UNSET:
            field_dict["conditionalVersion"] = conditional_version
        if fallback_greeting is not UNSET:
            field_dict["fallbackGreeting"] = fallback_greeting
        if key is not UNSET:
            field_dict["key"] = key
        if recipients is not UNSET:
            field_dict["recipients"] = recipients
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if status is not UNSET:
            field_dict["status"] = status
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_conditional_version import PlacementConditionalVersion # noqa: PLC0415
        from ..models.placement_language import PlacementLanguage # noqa: PLC0415
        from ..models.placement_test_value import PlacementTestValue # noqa: PLC0415
        from ..models.placement_variant import PlacementVariant # noqa: PLC0415
        d = dict(src_dict)
        conditional = []
        _conditional = d.pop("conditional")
        for conditional_item_data in (_conditional):
            conditional_item = PlacementConditionalVersion.from_dict(conditional_item_data)



            conditional.append(conditional_item)


        id = d.pop("id")

        kind = PlacementContentViewKind(d.pop("kind"))




        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = PlacementLanguage.from_dict(languages_item_data)



            languages.append(languages_item)


        name = d.pop("name")

        sensitive = cast(list[str], d.pop("sensitive"))


        test_data = cast(list[str], d.pop("testData"))


        test_values = []
        _test_values = d.pop("testValues")
        for test_values_item_data in (_test_values):
            test_values_item = PlacementTestValue.from_dict(test_values_item_data)



            test_values.append(test_values_item)


        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = PlacementVariant.from_dict(variants_item_data)



            variants.append(variants_item)


        audience = d.pop("audience", UNSET)

        category = d.pop("category", UNSET)

        conditional_version = d.pop("conditionalVersion", UNSET)

        fallback_greeting = d.pop("fallbackGreeting", UNSET)

        key = d.pop("key", UNSET)

        recipients = d.pop("recipients", UNSET)

        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        status = d.pop("status", UNSET)

        version = d.pop("version", UNSET)

        placement_content_view = cls(
            conditional=conditional,
            id=id,
            kind=kind,
            languages=languages,
            name=name,
            sensitive=sensitive,
            test_data=test_data,
            test_values=test_values,
            variants=variants,
            audience=audience,
            category=category,
            conditional_version=conditional_version,
            fallback_greeting=fallback_greeting,
            key=key,
            recipients=recipients,
            scheduled_at=scheduled_at,
            sent_at=sent_at,
            status=status,
            version=version,
        )

        return placement_content_view

