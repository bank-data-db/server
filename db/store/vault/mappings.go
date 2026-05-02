package store


func (s *DBStore) MappingInsert(ctx context.Context, authorID string, m *data.Mapping) (string, error) {
	id := snownode.NewID()

	_, err := s.db.Exec(
		ctx,
		`INSERT INTO mappings (
			id, author_id, name, priority,
			trans_text, trans_amount, res_name, res_category
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, authorID, m.Name, m.Priority,
		m.InpText.TextNil(), m.InpAmt, m.ResName, m.ResCategoryID,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}
